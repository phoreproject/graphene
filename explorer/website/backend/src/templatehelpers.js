const Handlebars = require('handlebars');
const fs = require('fs');

function doRegisterPartial(fileName)
{
	let name = fileName;
	name = name.substr(0, name.lastIndexOf('.')) || name;
	name = name.substr(name.lastIndexOf('/')) || name;
	Handlebars.registerPartial(name, fs.readFileSync(fileName, 'utf8'));
}

function registerPartials(application)
{
	doRegisterPartial(application.getFrontEndViewsPath('/partials/test.hbs'));
	doRegisterPartial(application.getFrontEndViewsPath('/partials/blocktable.hbs'));
	doRegisterPartial(application.getFrontEndViewsPath('/partials/blockdetail.hbs'));
}

function registerHelpers(application)
{
	Handlebars.registerHelper("url", function(url) {
		return url;
	});
}

function initializeTemplateHelpers(application)
{
	registerHelpers(application)

	registerPartials(application);
}

module.exports = { initializeTemplateHelpers: initializeTemplateHelpers };
