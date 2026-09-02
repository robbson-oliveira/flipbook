const fs = require('fs');
const path = require('path');

const sounds = [
  'https://cdnm.heyzine.com/flipbook/snd/flip-ct-sm.mp3',
  'https://cdnm.heyzine.com/flipbook/snd/flip-ct-md.mp3',
  'https://cdnm.heyzine.com/flipbook/snd/flip-ct-lg.mp3'
];

async function downloadSounds() {
  if (!fs.existsSync('sounds')) fs.mkdirSync('sounds');
  if (!fs.existsSync('web/sounds')) fs.mkdirSync('web/sounds', { recursive: true });

  for (const url of sounds) {
    const filename = path.basename(url);
    console.log('Downloading:', filename);
    const res = await fetch(url);
    const buffer = Buffer.from(await res.arrayBuffer());
    fs.writeFileSync(path.join('sounds', filename), buffer);
    fs.writeFileSync(path.join('web/sounds', filename), buffer);
    console.log('Saved:', filename, buffer.length, 'bytes. Base64 length:', buffer.toString('base64').length);
  }
}

downloadSounds();
