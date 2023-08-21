import { writable } from "svelte/store";

const Themes = {
	"default": 0,
	"UNSPECIFIED": 100,
} as const;

type Theme = keyof typeof Themes;

const theme = writable<Theme>("default");


const SCHEME_ATTRIBUTE = 'data-color-scheme';

const scheme = {
	dark: function () {
		document.documentElement.setAttribute(SCHEME_ATTRIBUTE, 'dark');
	},

	light: function () {
		document.documentElement.setAttribute(SCHEME_ATTRIBUTE, 'light');
	},

	onChange: function () {
		window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (event) => {
			const colorScheme = event.matches ? 'dark' : 'light';
			document.documentElement.setAttribute('data-color-scheme', colorScheme);
		});
	},
};

export {
	theme,
	scheme,
};

