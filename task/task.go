package task

import (
	"bufio"
	"os"
)

func StartTask(onData func(string), onError func(error)) {
	bufr := bufio.NewReader(os.Stdin)

	for {
		input, err := bufr.ReadString('\n')
		if err != nil {
			onError(err)
			break
		}

		onData(input)
	}
}
