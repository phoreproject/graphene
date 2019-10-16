const Controller = require('./controller').Controller;
const util = require('../util');

class ShardController extends Controller {
	constructor() {
		super();
	}

	getShards(request, response) {
		let urlQueries = request.query;
		let query = this.getDatabase().select().from('shards');
		if(urlQueries.i) {
			query = query.offset(parseInt(urlQueries.i));
		}
		if(urlQueries.c) {
			query = query.limit(parseInt(urlQueries.c));
		}
		query.then(function(shardList) {
			response.json(util.bufferFieldsToHex(shardList));
		});
	}

}

module.exports = { ShardController: ShardController };
