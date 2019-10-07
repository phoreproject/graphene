import jQuery from "jquery";
window.$ = window.jQuery = jQuery;

//import dt from 'datatables.net';

import 'datatables.net';
import 'datatables.net-zf';

//require('datatables.net-zf')();

window.util = require('./util');

window.sprintf = require('sprintf-js').sprintf;
window.vsprintf = require('sprintf-js').vsprintf;
