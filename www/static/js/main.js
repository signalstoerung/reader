console.log("main.js loaded");
const feedSelector = document.getElementById('feed');
feedSelector.addEventListener('change', (event) => {
  console.log(`Changing to feed: ${event.target.value}`);
  const feed = event.target.value;
  const location = window.location;
  location.search = `feed=${feed}`;
  location.reload;
});

function toggle(id) {
  const elem = document.getElementById(id);
  if (elem.style.display != "block") {
  elem.style.display = "block";
  } else {
  elem.style.display = "none";
  }
}