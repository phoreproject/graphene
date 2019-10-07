const Controller = require('./controller').Controller;

class BlockController extends Controller {
	constructor() {
		super();
	}

	getBlocks(request, response) {
		let urlQueries = request.query;
		let query = this.getDatabase().select().from('blocks');
		if(urlQueries.i) {
			query = query.offset(parseInt(urlQueries.i));
		}
		if(urlQueries.c) {
			query = query.limit(parseInt(urlQueries.c));
		}
		query.then(function(blockList) {
			response.json(blockList);
		});
	}

	getBlock(request, response) {
		let hash = request.params.hash;
		this.getDatabase().select().from('blocks').whereRaw('hex(hash) = ?', [ hash ]).then((blockList) => {
			if(blockList.length === 0) {
				this.apiSend404(response);
			}
			else {
				response.json(blockList[0]);
			}
		});
	}

}

module.exports = { BlockController: BlockController };
