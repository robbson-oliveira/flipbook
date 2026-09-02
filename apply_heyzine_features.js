const fs = require('fs');

const b64 = JSON.parse(fs.readFileSync('sounds_base64.json', 'utf8'));

let html = fs.readFileSync('index.html', 'utf8');

// Replace the audio synthesis with exact Heyzine base64 audio players
const oldAudioBlock = `    // ── Realistic Page Flip Sound Synthesis (Heyzine Style) ──
    let audioCtx = null;
    let soundEnabled = true;

    function getAudioContext() {
      if (!audioCtx) {
        audioCtx = new (window.AudioContext || window.webkitAudioContext)();
      }
      if (audioCtx && audioCtx.state === 'suspended') {
        audioCtx.resume();
      }
      return audioCtx;
    }

    function playPageFlipSound() {
      if (!soundEnabled) return;
      try {
        const ctx = getAudioContext();
        if (!ctx) return;
        const duration = 0.24;
        const sampleRate = ctx.sampleRate;
        const buffer = ctx.createBuffer(1, Math.floor(sampleRate * duration), sampleRate);
        const data = buffer.getChannelData(0);

        // Realistic multi-layer paper flutter and rustle
        for (let i = 0; i < buffer.length; i++) {
          const t = i / (sampleRate * duration);
          const envelope = Math.pow(Math.sin(t * Math.PI), 1.5) * Math.exp(-t * 2.5);
          const flutter = Math.sin(t * 90 + Math.sin(t * 30) * 8);
          data[i] = (Math.random() * 2 - 1) * envelope * (0.85 + 0.15 * flutter);
        }

        const source = ctx.createBufferSource();
        source.buffer = buffer;

        // Bandpass filter sweep for physical paper swish (1500Hz -> 380Hz)
        const filter = ctx.createBiquadFilter();
        filter.type = 'bandpass';
        filter.frequency.setValueAtTime(1500, ctx.currentTime);
        filter.frequency.exponentialRampToValueAtTime(380, ctx.currentTime + duration);
        filter.Q.setValueAtTime(1.8, ctx.currentTime);

        const gain = ctx.createGain();
        gain.gain.setValueAtTime(0.4, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration);

        source.connect(filter);
        filter.connect(gain);
        gain.connect(ctx.destination);

        source.start();
      } catch (e) {
        console.warn('Audio playback error:', e);
      }
    }`;

const newAudioBlock = `    // ── Exact Heyzine Audio Assets (Embedded Base64 for 0ms Latency) ──
    const HEYZINE_SOUNDS = {
      sm: 'data:audio/mp3;base64,${b64.sm}',
      md: 'data:audio/mp3;base64,${b64.md}',
      lg: 'data:audio/mp3;base64,${b64.lg}'
    };

    let audioPool = [];
    let soundEnabled = true;

    function playPageFlipSound(type = 'md') {
      if (!soundEnabled) return;
      try {
        const src = HEYZINE_SOUNDS[type] || HEYZINE_SOUNDS.md;
        const audio = new Audio(src);
        audio.volume = 0.75;
        audio.play().catch(() => {});
      } catch (e) {
        console.warn('Audio play error:', e);
      }
    }`;

html = html.replace(oldAudioBlock, newAudioBlock);

// Replace CSS for .page, .page-cover, spine shadow and stacked pages
const oldCssBlock = `    /* Page elements inside flipbook */
    .page {
      background-color: #ffffff;
      overflow: hidden;
      position: relative;
      border-radius: 2px;
      box-shadow:
        0 8px 30px rgba(0, 0, 0, 0.5),
        0 0 0 1px rgba(255, 255, 255, 0.08);
    }

    .page.page-cover {
      box-shadow:
        0 15px 50px rgba(0, 0, 0, 0.7),
        0 0 0 1px rgba(255, 255, 255, 0.12);
      border-radius: 4px;
    }
    .page-content {
      width: 100%;
      height: 100%;
      position: relative;
      display: flex;
      align-items: center;
      justify-content: center;
      background: #fff;
    }
    .page-content img {
      width: 100%;
      height: 100%;
      object-fit: fill;
      display: block;
      image-rendering: -webkit-optimize-contrast;
      image-rendering: high-quality;
      transform: translateZ(0);
      backface-visibility: hidden;
    }

    /* Spine and page shadows */
    .page.--left .page-content::after {
      content: '';
      position: absolute;
      top: 0; right: 0; bottom: 0;
      width: 25px;
      background: linear-gradient(to left, rgba(0,0,0,0.18), transparent);
      pointer-events: none;
    }
    .page.--right .page-content::after {
      content: '';
      position: absolute;
      top: 0; left: 0; bottom: 0;
      width: 25px;
      background: linear-gradient(to right, rgba(0,0,0,0.18), transparent);
      pointer-events: none;
    }`;

const newCssBlock = `    /* ─── Realistic 3D Stacked Pages & Spine (Heyzine Style) ─── */
    .page {
      background-color: #ffffff;
      overflow: hidden;
      position: relative;
      border-radius: 1px;
    }

    /* Cover Page (closed book): stacked page edges on right & top + deep cast shadow */
    .page.page-cover {
      border-radius: 2px 5px 5px 2px;
      box-shadow:
        0 0 0 1px rgba(0, 0, 0, 0.08),
        /* Layered stacked paper pages on right edge */
        1px 0px 0 #edebe6,
        2px 1px 0 #dfdbd0,
        3px 1px 0 #edebe6,
        4px 2px 0 #d3cebf,
        5px 2px 0 #edebe6,
        6px 3px 0 #c7c1b0,
        7px 3px 0 #edebe6,
        8px 4px 0 #b8b19e,
        9px 4px 0 #edebe6,
        10px 5px 0 #a9a28f,
        /* Deep ambient and directional drop shadows */
        14px 20px 40px rgba(0, 0, 0, 0.65),
        28px 40px 80px rgba(0, 0, 0, 0.5);
    }

    /* Cover Spine Shadow on left edge (intense 3D binding shadow + bevel highlight) */
    .page.page-cover .page-content::before {
      content: '';
      position: absolute;
      top: 0; left: 0; bottom: 0;
      width: 38px;
      background: linear-gradient(to right,
        rgba(0, 0, 0, 0.42) 0%,
        rgba(0, 0, 0, 0.22) 18%,
        rgba(255, 255, 255, 0.16) 32%,
        rgba(0, 0, 0, 0.12) 55%,
        rgba(0, 0, 0, 0.03) 80%,
        transparent 100%
      );
      z-index: 10;
      pointer-events: none;
      box-shadow: inset 3px 0 8px rgba(0, 0, 0, 0.35);
    }

    /* Open Left Page: stacked pages on left side, deep gutter crease on right */
    .page.--left {
      border-radius: 4px 1px 1px 4px;
      box-shadow:
        0 0 0 1px rgba(0, 0, 0, 0.06),
        -1px 0px 0 #edebe6,
        -2px 1px 0 #dfdbd0,
        -3px 1px 0 #edebe6,
        -4px 2px 0 #d3cebf,
        -5px 2px 0 #edebe6,
        -6px 3px 0 #c7c1b0,
        -12px 18px 40px rgba(0, 0, 0, 0.55);
    }
    .page.--left .page-content::after {
      content: '';
      position: absolute;
      top: 0; right: 0; bottom: 0;
      width: 48px;
      background: linear-gradient(to left, 
        rgba(0, 0, 0, 0.42) 0%,
        rgba(0, 0, 0, 0.2) 25%,
        rgba(0, 0, 0, 0.06) 65%,
        transparent 100%
      );
      pointer-events: none;
      z-index: 10;
    }

    /* Open Right Page: stacked pages on right side, deep gutter crease on left */
    .page.--right {
      border-radius: 1px 4px 4px 1px;
      box-shadow:
        0 0 0 1px rgba(0, 0, 0, 0.06),
        1px 0px 0 #edebe6,
        2px 1px 0 #dfdbd0,
        3px 1px 0 #edebe6,
        4px 2px 0 #d3cebf,
        5px 2px 0 #edebe6,
        6px 3px 0 #c7c1b0,
        12px 18px 40px rgba(0, 0, 0, 0.55);
    }
    .page.--right .page-content::after {
      content: '';
      position: absolute;
      top: 0; left: 0; bottom: 0;
      width: 48px;
      background: linear-gradient(to right, 
        rgba(0, 0, 0, 0.42) 0%,
        rgba(0, 0, 0, 0.2) 25%,
        rgba(0, 0, 0, 0.06) 65%,
        transparent 100%
      );
      pointer-events: none;
      z-index: 10;
    }

    /* Back Cover (last page) */
    .page.page-back {
      border-radius: 5px 2px 2px 5px;
      box-shadow:
        0 0 0 1px rgba(0, 0, 0, 0.08),
        -1px 0px 0 #edebe6,
        -2px 1px 0 #dfdbd0,
        -3px 1px 0 #edebe6,
        -4px 2px 0 #d3cebf,
        -5px 2px 0 #edebe6,
        -6px 3px 0 #c7c1b0,
        -7px 3px 0 #edebe6,
        -8px 4px 0 #b8b19e,
        -9px 4px 0 #edebe6,
        -10px 5px 0 #a9a28f,
        -14px 20px 40px rgba(0, 0, 0, 0.65),
        -28px 40px 80px rgba(0, 0, 0, 0.5);
    }
    .page.page-back .page-content::before {
      content: '';
      position: absolute;
      top: 0; right: 0; bottom: 0;
      width: 38px;
      background: linear-gradient(to left,
        rgba(0, 0, 0, 0.42) 0%,
        rgba(0, 0, 0, 0.22) 18%,
        rgba(255, 255, 255, 0.16) 32%,
        rgba(0, 0, 0, 0.12) 55%,
        transparent 100%
      );
      z-index: 10;
      pointer-events: none;
    }

    .page-content {
      width: 100%;
      height: 100%;
      position: relative;
      display: flex;
      align-items: center;
      justify-content: center;
      background: #fff;
    }
    .page-content img {
      width: 100%;
      height: 100%;
      object-fit: fill;
      display: block;
      image-rendering: -webkit-optimize-contrast;
      image-rendering: high-quality;
      transform: translateZ(0);
      backface-visibility: hidden;
    }`;

html = html.replace(oldCssBlock, newCssBlock);

// Also add page-back class to last page
html = html.replace("pageDiv.className = 'page' + (i === 1 ? ' page-cover' : '');", "pageDiv.className = 'page' + (i === 1 ? ' page-cover' : (i === TOTAL_PAGES ? ' page-back' : ''));");

// Trigger large flip sound for jumps (first/last page buttons)
html = html.replace("document.getElementById('btn-first').addEventListener('click', () => pageFlip && pageFlip.flip(0));", "document.getElementById('btn-first').addEventListener('click', () => { if (pageFlip) { playPageFlipSound('lg'); pageFlip.flip(0); } });");
html = html.replace("document.getElementById('btn-last').addEventListener('click', () => pageFlip && pageFlip.flip(pageFlip.getPageCount() - 1));", "document.getElementById('btn-last').addEventListener('click', () => { if (pageFlip) { playPageFlipSound('lg'); pageFlip.flip(pageFlip.getPageCount() - 1); } });");

fs.writeFileSync('index.html', html);
fs.writeFileSync('web/index.html', html);
console.log('Updated index.html and web/index.html successfully!');
