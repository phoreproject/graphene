const util = require('../util');

class Controller {
	constructor() {
	}

	initialize(application) {
		this.application = application;

		this.doInitialize();
	}

	getApplication() {
		return this.application;
	}

	getDatabase() {
		return this.application.getDatabase();
	}

	apiSend404(response) {
		response.status(404);
		response.json({
			message: 'not found'
		});
	}
	
	send404(response) {
		response.status(404);
		response.json({
			message: 'not found'
		});
	}
	
	apiSendError(response, errorMessage, errorCode) {
		errorMessage = util.checkValue(errorMessage, "Error");
		errorCode = util.checkValue(errorCode, 1);
		response.json({
			message: errorMessage,
			erroCode: errorCode
		});
	}
	
	doInitialize() {
	}
}

module.exports = { Controller: Controller };
