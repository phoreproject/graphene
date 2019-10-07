const Controller = require('./controller').Controller;

class TransactionController extends Controller {
	constructor() {
		super();
	}

	getTransactions(request, response) {
		let urlQueries = request.query;
		let query = this.getDatabase().select().from('transactions');
		if(urlQueries.i) {
			query = query.offset(parseInt(urlQueries.i));
		}
		if(urlQueries.c) {
			query = query.limit(parseInt(urlQueries.c));
		}
		query.then(function(transactionList) {
			response.json(transactionList);
		});
	}

	getTransaction(request, response) {
		let hash = request.params.hash;
		this.getDatabase().select().from('transactions').whereRaw('hex(recipient_hash) = ?', [ hash ]).then((transactionList) => {
			if(transactionList.length === 0) {
				this.apiSend404(response);
			}
			else {
				response.json(transactionList[0]);
			}
		});
	}

}

module.exports = { TransactionController: TransactionController };
