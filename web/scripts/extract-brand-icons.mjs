import fs from 'node:fs';
import path from 'node:path';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

import OpenAI from '/root/opencrab/web/node_modules/@lobehub/icons/es/OpenAI/components/Avatar.js';
import Claude from '/root/opencrab/web/node_modules/@lobehub/icons/es/Claude/components/Avatar.js';
import Gemini from '/root/opencrab/web/node_modules/@lobehub/icons/es/Gemini/components/Avatar.js';
import OpenRouter from '/root/opencrab/web/node_modules/@lobehub/icons/es/OpenRouter/components/Avatar.js';
import Moonshot from '/root/opencrab/web/node_modules/@lobehub/icons/es/Moonshot/components/Avatar.js';
import Minimax from '/root/opencrab/web/node_modules/@lobehub/icons/es/Minimax/components/Avatar.js';
import Zhipu from '/root/opencrab/web/node_modules/@lobehub/icons/es/Zhipu/components/Avatar.js';

const icons = {
  openai: OpenAI,
  claude: Claude,
  gemini: Gemini,
  openrouter: OpenRouter,
  moonshot: Moonshot,
  minimax: Minimax,
  zhipu: Zhipu,
};

const outDir = '/root/opencrab/web/public/brands';
fs.mkdirSync(outDir, { recursive: true });

for (const [name, Icon] of Object.entries(icons)) {
  const markup = renderToStaticMarkup(React.createElement(Icon, { size: 24 }));
  fs.writeFileSync(path.join(outDir, `${name}.svg`), `${markup}\n`);
}

console.log(`wrote ${Object.keys(icons).length} svg files`);
