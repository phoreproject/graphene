const util = {};

util.toHexString = function(byteArray) {
  return Array.from(byteArray, function(byte) {
    return ('0' + (byte & 0xFF).toString(16)).slice(-2);
  }).join('')
}

util.shortenStringWithEllipsis = function(text)
{
  if(text.length < 12) {
    return text
  }

  return text.substring(0, 4)
    + '...'
    + text.substring(text.length - 4)
}

module.exports = util;
