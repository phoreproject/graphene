const util = require('./util');

class Application {
	constructor() {
		this.express = require('express')();
		this.config = require('config');

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

	getExpress() {
		return this.express;
	}

	initialize() {
		this.doInitializeDatabase();

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

	run() {
		const serverConfig = this.config.get('server');
		const port = util.checkValue(serverConfig.port, 8080);
		this.express.listen(port, () => {
			console.log("Server is listening on port " + port);
		});
	}
}

module.exports = { Application: Application };
