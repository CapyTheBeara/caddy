process.stdin.setEncoding('utf8');

process.stdin.on('data', function(data) {
  eval(data);
});
