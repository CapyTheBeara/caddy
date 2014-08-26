process.stdin.setEncoding('utf8');

process.stdin.on('data', function(chunk) {
  eval(chunk);
});
