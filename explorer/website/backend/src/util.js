const util = {};

util.hexToString = function(hex)
{
    let str = '';
    for(let i = 0; i < hex.length; i += 2) {
		str += String.fromCharCode(parseInt(hex.substr(i, 2), 16));
	}
    return str;	
}

util.stringToHex = function(str)
{
	let hex = '';
	for (let i = 0; i < str.length; ++i) {
		hex += Number(str.charCodeAt(i)).toString(16);
	}
	return hex;
}

util.checkValue = function(value, defaultValue)
{
	if(value === undefined) {
		if(defaultValue === undefined) {
			defaultValue = null;
		}
		return defaultValue;
	}

	return value;
}

util.bufferFieldsToHex = function(dict)
{
	if(Array.isArray(dict)) {
		for(var i = 0; i < dict.length; ++i) {
			dict[i] = util.bufferFieldsToHex(dict[i]);
		}
	}
	else {
		for(var key in dict) {
			if(dict[key] && dict[key].constructor == Buffer) {
				dict[key] = dict[key].toString('hex');
			}
		}
	}

	return dict;
}

module.exports = util;
