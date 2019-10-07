const util = require('./util');

class Application {
	constructor() {
		this.expressClass = require('express');
		this.express = this.expressClass();
		this.config = require('config');

		this.frontEndPath = __dirname + '/../../frontend';

		this.controllerMap = {};

		this.initialize();
	}

	getController(controllerClass) {
		const name = controllerClass.name;
		if(! this.controllerMap[name]) {
			 const controller = new controllerClass();
			 this.controllerMap[name] = controller;
			 controller.initialize(this);
		}
		return this.controllerMap[name];
	}

	getControllerRoute(controllerClass, functionName) {
		const controller = this.getController(controllerClass);
		return controller[functionName].bind(controller);
	}

	getDatabase() {
		return this.database;
	}

	getExpressClass() {
		return this.expressClass;
	}

	getExpress() {
		return this.express;
	}

	getFrontEndPath() {
		return this.frontEndPath;
	}

	getFrontEndViewsPath(path) {
		if(! path) {
			path = '';
		}
		return this.frontEndPath + '/views' + path;
	}

	initialize() {
		this.doInitializeDatabase();
		this.doInitializeTemplateEngine();

		require('./routes').initializeRoutes(this);
	}

	doInitializeDatabase() {
		const dbConfig = this.config.get('database');
		const knex = require('knex');
		this.database = knex({
			client: dbConfig.driver,
			connection: {
				host: dbConfig.host,
				user: dbConfig.user,
				password: dbConfig.password,
				database: dbConfig.database,
			}
		});
		//this.database.on('query', console.log);
	}

	doInitializeTemplateEngine() {
		const hbs = require('express-handlebars');
		this.express.set('view engine', 'hbs');

		const viewDirName = this.getFrontEndViewsPath();
		this.express.set('views', viewDirName);
		this.express.engine('hbs', hbs({
			extname: 'hbs',
			defaultView: 'home',
			defaultLayout: 'default',
			layoutsDir: viewDirName + '/layouts/',
			partialsDir: viewDirName + '/partials/'
		}));

		require('./templatehelpers').initializeTemplateHelpers(this);
	}

	run() {
		const serverConfig = this.config.get('server');
		const port = util.checkValue(serverConfig.port, 8080);
		this.express.listen(port, () => {
			console.log("Server is listening on port " + port);
		});
	}
}

module.exports = { Application: Application };
