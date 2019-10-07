const Controller = require('./controller').Controller;
const util = require('../util');

class BlockController extends Controller {
	constructor() {
		super();
	}

	viewBlockDetail(request, response) {
		const urlQueries = request.query;
		const hash = urlQueries.hash;
		this.getDatabase().select().from('blocks').whereRaw('hex(hash) = ?', [ hash ]).then((blockList) => {
			if(blockList.length === 0) {
				this.send404(response);
			}
			else {
				const block = util.bufferFieldsToHex(blockList[0]);
				response.render('pages/blockdetail', { title: 'Block detail', block: block });
			}
		});
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
			response.json(util.bufferFieldsToHex(blockList));
		});
	}

	getBlock(request, response) {
		let hash = request.params.hash;
		this.getDatabase().select().from('blocks').whereRaw('hex(hash) = ?', [ hash ]).then((blockList) => {
			if(blockList.length === 0) {
				this.apiSend404(response);
			}
			else {
				response.json(util.bufferFieldsToHex(blockList[0]));
			}
		});
	}

}

module.exports = { BlockController: BlockController };
