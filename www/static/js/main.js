console.log("main.js loaded");
const feedSelector = document.getElementById('feed');
feedSelector.addEventListener('change', (event) => {
  console.log(`Changing to feed: ${event.target.value}`);
  const feed = event.target.value;
  const location = window.location;
  location.search = `feed=${feed}`;
  location.reload;
});

const headlineCountItem = document.getElementById("headlineCount");
const headlineCount = headlineCountItem.getAttribute("data-count");
console.log(`Headline count: ${headlineCount}`);
addListeneners();

function addListeneners() {
  for (var i=0; i < headlineCount; i++) {
    const lnk = "lnk" + i;
    const prv = "prv" + i;
    const link = document.getElementById(lnk);
    link.addEventListener("click", (event)=>{
      const preview = document.getElementById(prv);
      if (preview.style.display != "block") {
        preview.style.display = "block";
      } else {
        preview.style.display = "none";
      }
      event.preventDefault();
    })
  }  
}