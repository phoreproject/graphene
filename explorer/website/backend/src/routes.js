const bodyParser = require('body-parser');

const BlockController = require('./controllers/blockcontroller').BlockController;
const TransactionController = require('./controllers/transactioncontroller').TransactionController;
const ValidatorController = require('./controllers/validatorcontroller').ValidatorController;
const ShardController = require('./controllers/shardcontroller').ShardController;

function initializeRoutes(application) {
	const _c = function(controllerClass, functionName) {
		return application.getControllerRoute(controllerClass, functionName);
	};

	const express = application.getExpress();

	// parse requests of content-type - application/x-www-form-urlencoded
	express.use(bodyParser.urlencoded({ extended: true }));

	// parse requests of content-type - application/json
	express.use(bodyParser.json());

	const publicFolder =  __dirname + '/../../frontend/public/';
	express.use('/', application.getExpressClass().static(publicFolder));
	
	express.get('/', (request, response) => {
		response.render('pages/home', { title: 'Home page' });
	});
	
	express.get('/validators', (request, response) => {
		response.render('pages/validators', { title: 'Validator page' });
	});
	
	express.get('/shards', (request, response) => {
		response.render('pages/shards', { title: 'Shard page' });
	});
	
	express.get('/block', _c(BlockController, 'viewBlockDetail'));

	express.get('/api/blocks', _c(BlockController, 'getBlocks'));
	express.get('/api/blocks/:hash', _c(BlockController, 'getBlock'));

	express.get('/api/transactions', _c(TransactionController, 'getTransactions'));
	express.get('/api/transactions/:hash', _c(TransactionController, 'getTransaction'));

	express.get('/api/validators', _c(ValidatorController, 'getValidators'));
	express.get('/api/validators/:hash', _c(ValidatorController, 'getValidator'));
	
	express.get('/api/shards', _c(ShardController, 'getShards'));

	express.get('/test', function (req, res) {
		res.send(res.__('test'));
	});	
}

module.exports = { initializeRoutes: initializeRoutes };
