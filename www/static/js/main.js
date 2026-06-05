const feedSelector = document.getElementById('feed');
const searchField = document.getElementById('searchTerms');
const searchButton = document.getElementById('searchButton');

if (feedSelector) {
  feedSelector.addEventListener('change', (event) => {
    const feed = event.target.value;
    const location = window.location;
    const params = new URL(location).searchParams;
    const search = params.get('q');
    if (search) {
      location.search = `feed=${encodeURIComponent(feed)}&q=${encodeURIComponent(search)}`;
    } else {
      location.search = `feed=${encodeURIComponent(feed)}`;
    }
  });
}

function redirect(searchTerms) {
  const location = window.location;
  const params = new URL(location).searchParams;
  const feed = params.get('feed');
  if (feed) {
    location.search = `?q=${encodeURIComponent(searchTerms)}&feed=${encodeURIComponent(feed)}`;
  } else {
    location.search = `q=${encodeURIComponent(searchTerms)}`;
  }
}

if (searchField) {
  searchField.addEventListener('keydown', (event) => {
    if (event.key === "Enter") {
      redirect(searchField.value);
    }
  });
}

if (searchButton && searchField) {
  searchButton.addEventListener('click', () => {
    redirect(searchField.value);
  });
}

function toggleArticleAsides() {
  const articles = document.querySelectorAll('.ticker-item');

  articles.forEach(article => {
    const headlineToggle = article.querySelector('.headline-toggle');
    const preview = article.querySelector('.headline-preview');

    if (headlineToggle && preview) {
      headlineToggle.addEventListener('click', function(event) {
        event.preventDefault();
        preview.style.display = preview.style.display === "block" ? "none" : "block";
      });
    }

    const save = article.querySelector('.saveAction');
    if (save) {
      save.addEventListener("click", () => {
        const itemId = save.dataset.id;
        const formData = new FormData();
        formData.append('action', 'save');
        formData.append('itemId', itemId);

        fetch('/saved/', {
          method: 'POST',
          body: formData
        })
        .then(response => {
          if (!response.ok) {
            save.classList.add('save-error');
            save.textContent = 'Error';
          } else {
            save.classList.add('saved');
            save.textContent = 'Saved';
          }
        });
      });
    }

    const share = article.querySelector('.shareAction');
    if (share && navigator.share) {
      share.addEventListener("click", (event) => {
        event.preventDefault();
        navigator.share({
          title: share.dataset.headline,
          text: share.dataset.headline + "\n" + share.dataset.preview,
          url: share.dataset.link
        });
      });
    } else if (share) {
      share.hidden = true;
    }
  });
}

toggleArticleAsides();
