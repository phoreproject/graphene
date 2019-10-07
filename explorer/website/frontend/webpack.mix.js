let mix = require('laravel-mix');

const publicFolder = 'public/';

mix.copy('images', publicFolder + '/images');

mix
	.js('js/app.js', publicFolder + 'js/')
	.sass('sass/app.scss', publicFolder + 'css/')
;

