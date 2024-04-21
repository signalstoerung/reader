function generateWobblyLine(length, wobbleAmount) {
    const startX = 0;
    const startY = 10;
    const lineWidth = 15;
    let path = `M ${startX},${startY} `;
    const distanceX = length;
    const segments = Math.floor(distanceX / 30) + 1; // Adjust for more/less segments
  
    for (let i = 0; i < segments; i++) {
      const offsetY = Math.random() * wobbleAmount * (Math.random() < 0.5 ? -1 : 1); // Random wobble value
      const x = startX + i * (distanceX / (segments - 1));
      const y = startY + offsetY;
      path += `L ${x},${y} `;
    }
    
    const svgElement = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svgElement.setAttribute('viewBox', `0 0 ${length} ${startY + wobbleAmount + lineWidth/2}`)
    const pathElem = document.createElementNS('http://www.w3.org/2000/svg', 'path');
    pathElem.setAttribute('d', path);
    pathElem.setAttribute('fill', 'none');
    pathElem.setAttribute('stroke', 'black');
    pathElem.setAttribute('stroke-width', lineWidth);
    
    svgElement.appendChild(pathElem);
    return svgElement;
  }
  
  // Example usage
  const wobbleAmount = 1; // Adjust for more/less wobble
  const headlines = Array.from(document.getElementsByClassName("redacted"));
  for (var headline of headlines) {
    console.log("Headline: " + headline);
    const headlineLink = Array.from(headline.getElementsByTagName("a"));
    if (headlineLink.length < 1) {
        continue;
    }
    const headlineWidth = headlineLink[0].getBoundingClientRect().width;
    const svg = generateWobblyLine(headlineWidth, wobbleAmount);
    svg.setAttribute('width', headlineWidth);
    headline.appendChild(svg);
  }