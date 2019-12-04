import jQuery from "jquery";
window.$ = window.jQuery = jQuery;

//import dt from 'datatables.net';

import 'datatables.net';
import 'datatables.net-zf';

//require('datatables.net-zf')();

window.util = require('./util');

window.sprintf = require('sprintf-js').sprintf;
window.vsprintf = require('sprintf-js').vsprintf;

$(function() {
	$(document).foundation();
});

var defaultTableOptions = {
	searching: false,
	lengthChange: false,
	responsive: true,
	fixedHeader: {
		header: true,
	},
	lengthMenu: [ [ 25, 50, 100 ], [ 25, 50, 100 ] ],
	pageLength: 25,
	info: false,
	headerCallback: function( thead, data, start, end, display ) {
		$(thead).closest('thead').find('th').each(function(){
		  //$(this).css('color', 'red').css('border','none');
		  $(this).css('border','none');
		});
	},
	initComplete: function(settings) {
		var table = settings.oInstance.api(); 
		$(table.table().node()).removeClass('no-footer');
	},
};

window.createTableOptions = function(options) {
	return Object.assign({}, options, defaultTableOptions)
};

