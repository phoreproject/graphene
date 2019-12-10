const Controller = require('./controller').Controller;
const util = require('../util');

class ValidatorController extends Controller {
	constructor() {
		super();
	}

	getValidators(request, response) {
		let urlQueries = request.query;
		let query = this.getDatabase().select().from('validators');
		if(urlQueries.start > 0) {
			query = query.offset(parseInt(urlQueries.start));
		}
		if(urlQueries.length) {
			query = query.limit(parseInt(urlQueries.length));
		}
		query.then((validatorList) => {
			this.getDatabase().count('id as C').from('validators').then(function(total) {
				let count = total[0].C;
				let returnData = {
					draw: query.draw ^ 0,
					recordsTotal: count,
					recordsFiltered: count,
					data: util.bufferFieldsToHex(validatorList),
				};
				response.json(returnData);
			});
		});
	}

	getValidator(request, response) {
		let hash = request.params.hash;
		this.getDatabase().select().from('validators').whereRaw('hex(validator_hash) = ?', [ hash ]).then((validatorList) => {
			if(validatorList.length === 0) {
				this.apiSend404(response);
			}
			else {
				response.json(util.bufferFieldsToHex(validatorList[0]));
			}
		});
	}

}

module.exports = { ValidatorController: ValidatorController };
