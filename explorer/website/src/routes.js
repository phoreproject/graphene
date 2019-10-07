const bodyParser = require('body-parser');

const BlockController = require('./controllers/blockcontroller').BlockController;
const TransactionController = require('./controllers/transactioncontroller').TransactionController;
const ValidatorController = require('./controllers/validatorcontroller').ValidatorController;

function initializeRoutes(application) {
	const _c = function(controllerClass, functionName) {
		return application.getControllerRoute(controllerClass, functionName);
	};

	const express = application.getExpress();

	// parse requests of content-type - application/x-www-form-urlencoded
	express.use(bodyParser.urlencoded({ extended: true }));

	// parse requests of content-type - application/json
	express.use(bodyParser.json());

	//express.use('/', express.static('public'));

	express.get('/', (request, response) => {
		response.json({"message": "Hello"});
	});
	express.get('/api/blocks', _c(BlockController, 'getBlocks'));
	express.get('/api/blocks/:hash', _c(BlockController, 'getBlock'));

	express.get('/api/transactions', _c(TransactionController, 'getTransactions'));
	express.get('/api/transactions/:hash', _c(TransactionController, 'getTransaction'));

	express.get('/api/validators', _c(ValidatorController, 'getValidators'));
	express.get('/api/validators/:hash', _c(ValidatorController, 'getValidator'));
}

module.exports = { initializeRoutes: initializeRoutes };
