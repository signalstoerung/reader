package newsticker

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/signalstoerung/reader/internal/feeds"
)

type Configuration struct {
	tickerChannel chan feeds.Item
	consumers     map[string]chan feeds.Item
}

var Config Configuration
var connectionMutex sync.Mutex

func (c *Configuration) SetTickerChannel(ch chan feeds.Item) {
	c.tickerChannel = ch
	c.consumers = make(map[string]chan feeds.Item)
}

func (c *Configuration) RegisterConsumer(name string, ch chan feeds.Item) {
	connectionMutex.Lock()
	defer connectionMutex.Unlock()
	c.consumers[name] = ch
}

func (c *Configuration) UnregisterConsumer(name string) {
	connectionMutex.Lock()
	defer connectionMutex.Unlock()
	delete(c.consumers, name)
}

func (c *Configuration) ConsumerExists(name string) bool {
	connectionMutex.Lock()
	defer connectionMutex.Unlock()
	_, ok := c.consumers[name]
	return ok
}

func ConsumeTicker(cancel chan struct{}) {
	if Config.tickerChannel == nil {
		log.Fatal("TICKER channel not set.")
	}
	log.Println("TICKER started.")
	for {
		select {
		case <-cancel:
			log.Print("TICKER cancelled.")
			return
		case item := <-Config.tickerChannel:
			log.Printf("TICKER item %v received.", item.Title)
			connectionMutex.Lock()
			for id, ch := range Config.consumers {
				select {
				case ch <- item:
					log.Printf("TICKER item %v sent to consumer %v.", item.Title, id)
				default:
					log.Printf("TICKER consumer %v is too slow/blocked, skipping item.", id)
				}
			}
			connectionMutex.Unlock()
		}
	}
}

func SimulateTicker(cancel chan struct{}) {
	tags := []string{"BREAKING: ", "BORING: ", "Here goes nothing: "}
	actions := []string{"reveals", "discovers", "unveils", "announces", "confirms", "denies", "admits", "claims", "welcomes", "proves", "disproves"}
	subjects := []string{"Area man", "Unhappy dog", "Cute ginger cat", "Alien", "Marginally famous politician", "'Teen Vogue' Celebrity", "AI", "Humanoid robot"}
	objects := []string{"fraud", "superhero powers", "uncanny genius for gutter cleaning", "robot overlord attack", "World War III", "alien invasion", "zombie apocalypse", "vampire romance", "AI uprising", "celebrity scandal"}
	if Config.tickerChannel == nil {
		log.Fatal("TICKER channel not set.")
	}
	log.Println("TICKER simulator started.")
	// create 20 items and send them to the ticker channel 5 seconds apart
	t := time.NewTicker(15 * time.Second)
outerloop:
	for {
		select {
		case <-t.C:
			// construct random headline using tags, actions, subjects, objects
			headline := tags[rand.Intn(len(tags))] + subjects[rand.Intn(len(subjects))] + " " + actions[rand.Intn(len(actions))] + " " + objects[rand.Intn(len(objects))]
			now := time.Now()
			Config.tickerChannel <- feeds.Item{Title: headline, Link: "https://example.com", PublishedParsed: &now}
		case <-cancel:
			break outerloop
		}
	}
	t.Stop()
	log.Println("TICKER simulator finished.")
}
