package main

type _Response struct {
	data string
	err  error
}

type NonBlock struct {
	chReq chan struct{}
	chRes chan _Response
}

func NewNonBlock(getter func() (string, error)) *NonBlock {
	chReq := make(chan struct{})
	chRes := make(chan _Response)

	go func() {
		for _ = range chReq {
			data, err := getter()
			chRes <- _Response{data: data, err: err}
		}
		close(chRes)
	}()

	return &NonBlock{
		chReq: chReq,
		chRes: chRes,
	}
}

func (w *NonBlock) GetOr(work func()) (string, error) {
	w.chReq <- struct{}{}
	for {
		select {
		case res := <-w.chRes:
			return res.data, res.err
		default:
			work()
		}
	}
}

func (w *NonBlock) Close() {
	close(w.chReq)
}
