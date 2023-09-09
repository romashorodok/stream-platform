import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import path from 'node:path';

const root = path.resolve(__dirname, "./")
const gen = path.join(root, "./src/gen/js")

// Intresting way of resolving  
// 
// import { createRequire } from 'module'
// const { resolve } = createRequire(import.meta.url)
// const prismaClient = `prisma${path.sep}client`
// const prismaClientIndexBrowser = resolve('@prisma/client/index-browser').replace(`@${prismaClient}`, `.${prismaClient}`)
// export default defineConfig(() => ({
//    resolve: { alias: { '.prisma/client/index-browser': path.relative(__dirname, prismaClientIndexBrowser) } },
// }))
//  the code result is
// resolve: {
//   alias: {
//     ".prisma/client/index-browser": "./node_modules/.prisma/client/index-browser.js"
//   }
// }


export default defineConfig({
	plugins: [sveltekit()],
	resolve: {
		alias: {
			'$gen': gen,
		},
	},
});
