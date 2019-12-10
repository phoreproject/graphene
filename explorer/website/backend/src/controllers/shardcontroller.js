const Controller = require('./controller').Controller;
const util = require('../util');

class ShardController extends Controller {
	constructor() {
		super();
	}

	getShards(request, response) {
		let urlQueries = request.query;
		let query = this.getDatabase().select().from('shards');
		if(urlQueries.start > 0) {
			query = query.offset(parseInt(urlQueries.start));
		}
		if(urlQueries.length) {
			query = query.limit(parseInt(urlQueries.length));
		}
		query.then((shardList) => {
			this.getDatabase().count('id as C').from('shards').then(function(total) {
				let count = total[0].C;
				let returnData = {
					draw: query.draw ^ 0,
					recordsTotal: count,
					recordsFiltered: count,
					data: util.bufferFieldsToHex(shardList),
				};
				response.json(returnData);
			});
		});
	}

}

module.exports = { ShardController: ShardController };
