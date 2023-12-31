package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	chatCompletionEndpoint = "https://api.openai.com/v1/chat/completions"
	ModelGPT3Latest        = "gpt-3.5-turbo-1106"
	ModelGPT3Standard      = "gpt-3.5-turbo"
	ModelGPT4Latest        = "gpt-4-1106-preview"
	ModelGPT4Standard      = "gpt-4"
	ResponseFormatJson     = "json_object"
	NewsEditorPrompt       = "The user will provide a list of headlines. You have the role of a breaking news editor. Review headlines and decide if they are highly newsworthy and should be sent via a push notification to a global audience. The bar is high. Consider only hard news. Ignore vague headlines, news analysis and opinion pieces. Be extremely critical and look for global relevance and high impact. For headlines that qualify, return a JSON object with a `news` property, which is array of objects that have an `ID` and `headline` field (both copied from the input), a `confidence` field (0-100) and a `reason` field (concise, in a few words)."
	FinishReasonMaxLength  = "length"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// dollars per 1000 tokens
var pricing = map[string]float64{
	"gpt-3.5-turbo":      0.0020,
	"gpt-3.5-turbo-1106": 0.0020,
	"gpt-4":              0.06,
	"gpt-4-1106-preview": 0.03,
}

type OpenAIApiStats struct {
	gorm.Model
	ApiKey       string `gorm:"-"`
	TokenCounter int
	CostCounter  float64
}

func (s *OpenAIApiStats) LogCostAndTokens(tokens int, cost float64) {
	s.TokenCounter += tokens
	s.CostCounter += cost
	db.Save(s)
}

var Stats OpenAIApiStats
var db *gorm.DB
var NewsContext string

type Message struct {
	Name    string `json:"name,omitempty"`
	Content string `json:"content"`
	Role    Role   `json:"role"`
}

type Request struct {
	Messages       []Message      `json:"messages"` // required
	Model          string         `json:"model"`    // required
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat ResponseFormat `json:"response_format,omitempty"`
}

// use the ResponseFormatJson const for Type
type ResponseFormat struct {
	Type string `json:"type"`
}

type CompletionChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Completion struct {
	Id      string             `json:"id"`
	Choices []CompletionChoice `json:"choices"`
	Created int                `json:"created"`
	Model   string             `json:"model"`
	Usage   Usage              `json:"usage"`
}

// calculate cost for completion
func (c Completion) Cost() (float64, error) {
	price, ok := pricing[c.Model]
	if !ok {
		return 0, fmt.Errorf("no price found for model %s", c.Model)
	}
	return price * float64(c.Usage.TotalTokens) / 1000, nil
}

func init() {
	var err error
	db, err = gorm.Open(sqlite.Open("./db/apistats.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	db.AutoMigrate(&OpenAIApiStats{})
	result := db.FirstOrCreate(&Stats)
	if result.Error != nil {
		log.Println(result.Error)
	}
}

func chatCompletion(request Request) (Completion, error) {
	completion := Completion{}
	reqBody, err := json.Marshal(request)
	//log.Printf("Json request object: %+v", string(reqBody))
	if err != nil {
		return completion, err
	}
	client := http.Client{}
	r, err := http.NewRequest(http.MethodPost, chatCompletionEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return completion, err
	}

	r.Header.Set("Authorization", "Bearer "+Stats.ApiKey)
	r.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(r)
	if err != nil {
		return completion, err
	}

	err = json.NewDecoder(resp.Body).Decode(&completion)
	if err != nil {
		return completion, err
	}

	if resp.StatusCode != http.StatusOK {
		return completion, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, resp.Status)
	}

	cost, err := completion.Cost()
	if err != nil {
		log.Printf("Completion generated, %v tokens (%v)", completion.Usage.TotalTokens, err)
		Stats.LogCostAndTokens(completion.Usage.TotalTokens, 0.0)
	} else {
		log.Printf("Completion generated at cost of $%.4f", cost)
		Stats.LogCostAndTokens(completion.Usage.TotalTokens, cost)
	}
	return completion, nil
}

// this should return a string of valid JSON
func ScoreHeadlines(text string) (string, error) {
	context := fmt.Sprintf("\nContext: \nToday is %v.", time.Now().Format("Jan 2, 2006"))
	if NewsContext != "" {
		context += " Long-running stories in the media currently include:\n"
		context += NewsContext
		log.Printf("Using news context: %v", context)
	}
	request := Request{
		Model:          ModelGPT3Latest,
		MaxTokens:      1200,
		ResponseFormat: ResponseFormat{Type: ResponseFormatJson},
		Messages: []Message{
			{
				Role:    RoleSystem,
				Content: NewsEditorPrompt + context,
			},
			{
				Role:    RoleUser,
				Content: text,
			},
		},
	}

	completion, err := chatCompletion(request)
	if err != nil {
		return "", err
	}
	choice := completion.Choices[0]
	if choice.FinishReason == FinishReasonMaxLength {
		log.Printf("WARNING: Chat completion exhausted maximum token length: %v", completion.Usage.TotalTokens)
	}
	return choice.Message.Content, nil
}
