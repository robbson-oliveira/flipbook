const fs = require('fs');

async function main() {
  const res = await fetch('https://heyzine.com/flip-book/5492f89fcc.html', {
    headers: {
      'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
    }
  });
  const html = await res.text();
  console.log('HTML length:', html.length);
  
  // Find sound references
  const audioMatches = [...html.matchAll(/["']([^"']*\.(?:mp3|ogg|wav|m4a)[^"']*)["']/gi)].map(m => m[1]);
  console.log('Audio matches in HTML:', audioMatches);

  const scriptMatches = [...html.matchAll(/<script[^>]+src=["']([^"']+)["']/gi)].map(m => m[1]);
  console.log('Scripts:', scriptMatches);

  for (const s of scriptMatches) {
    const sUrl = s.startsWith('http') ? s : 'https://heyzine.com' + (s.startsWith('/') ? '' : '/') + s;
    console.log('Checking script:', sUrl);
    try {
      const sRes = await fetch(sUrl);
      const sText = await sRes.text();
      const sAudio = [...sText.matchAll(/["']([^"']*\.(?:mp3|ogg|wav|m4a))["']/gi)].map(m => m[1]);
      if (sAudio.length > 0) {
        console.log('Found in script ' + sUrl + ':', sAudio);
      }
      // Check for base64 audio
      const base64Audio = [...sText.matchAll(/data:audio\/(?:mp3|wav|ogg|mpeg);base64,[A-Za-z0-9+/=]+/g)].map(m => m[0]);
      if (base64Audio.length > 0) {
        console.log('Found base64 audio in script:', base64Audio.length, 'items');
      }
    } catch(e) {
      console.log('Error fetching script:', e.message);
    }
  }
}

main();
