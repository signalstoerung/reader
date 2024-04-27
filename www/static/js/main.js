console.log("main.js loaded");
const feedSelector = document.getElementById('feed');
feedSelector.addEventListener('change', (event) => {
  console.log(`Changing to feed: ${event.target.value}`);
  const feed = event.target.value;
  const location = window.location;
  location.search = `feed=${feed}`;
  location.reload;
});

function toggleArticleAsides() {
  // get all <article> elements
  const articles = document.querySelectorAll('article');

  // do setup for each <article>
  articles.forEach(article => {
    // Setting up toggling the preview
    // get the link inside the .headline div
    const headlineLink = article.querySelector('.headline a');

    // add event listener to it
    headlineLink.addEventListener('click', function(event) {
      event.preventDefault(); // Prevent default link behavior

      const preview = article.querySelector('aside'); // this is the preview element
      if (preview.style.display != "block") {
        preview.style.display = "block"; // if display is not block, set it to block
      } else {
        preview.style.display = "none"; // else (if it is block), set it to hidden
      }
    }); // end event listener preview

    // set up 'save' action
    const save = article.querySelector('.saveAction');
    save.addEventListener("click", (event)=>{
      const itemId = save.dataset.id;

      // construct form data
      const formData = new FormData();
      formData.append('action', 'save');
      formData.append('itemId', itemId);
      
      // send POST request to /saved/ endpoint
      fetch('/saved/', {
          method: 'POST',
          body: formData
      })
      .then(response => {
          if (!response.ok) {
              save.style.color = 'red';
          } else {
            save.innerHTML = '&#9733;';
          }
      });    
    });

    // set up 'share' action
    const share = article.querySelector('.shareAction')
    share.addEventListener("click", (event)=>{
      event.preventDefault();
      navigator.share({
        title: share.dataset.headline,
        text: share.dataset.headline + "\n" + share.dataset.preview,
        url: share.dataset.link
      });
    });


  }); // end article loop
}

toggleArticleAsides(); 
