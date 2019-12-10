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
		if(urlQueries.start > 0) {
			query = query.offset(parseInt(urlQueries.start));
		}
		if(urlQueries.length) {
			query = query.limit(parseInt(urlQueries.length));
		}
		query.then((blockList) => {
			this.getDatabase().count('id as C').from('blocks').then(function(total) {
				let count = total[0].C;
				let returnData = {
					draw: query.draw ^ 0,
					recordsTotal: count,
					recordsFiltered: count,
					data: util.bufferFieldsToHex(blockList),
				};
				response.json(returnData);
			});
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
