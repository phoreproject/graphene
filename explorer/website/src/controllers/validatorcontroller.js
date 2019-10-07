const Controller = require('./controller').Controller;

class ValidatorController extends Controller {
	constructor() {
		super();
	}

	getValidators(request, response) {
		let urlQueries = request.query;
		let query = this.getDatabase().select().from('validators');
		if(urlQueries.i) {
			query = query.offset(parseInt(urlQueries.i));
		}
		if(urlQueries.c) {
			query = query.limit(parseInt(urlQueries.c));
		}
		query.then(function(validatorList) {
			response.json(validatorList);
		});
	}

	getValidator(request, response) {
		let hash = request.params.hash;
		this.getDatabase().select().from('validators').whereRaw('hex(validator_hash) = ?', [ hash ]).then((validatorList) => {
			if(validatorList.length === 0) {
				this.apiSend404(response);
			}
			else {
				response.json(validatorList[0]);
			}
		});
	}

}

module.exports = { ValidatorController: ValidatorController };
