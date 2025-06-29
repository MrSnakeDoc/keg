package prompter

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Prompter interface {
	Confirm(question string) (bool, error)
	Prompt(question string) (string, error)
}

type TextPrompter struct {
	in  *bufio.Reader
	out io.Writer
}

func New(in io.Reader, out io.Writer) *TextPrompter {
	return &TextPrompter{
		in:  bufio.NewReader(in),
		out: out,
	}
}

func (p *TextPrompter) Confirm(q string) (bool, error) {
	if _, err := fmt.Fprintf(p.out, "%s [y/N]: ", q); err != nil {
		return false, err
	}

	resp, err := p.in.ReadString('\n')
	if err != nil {
		return false, err
	}

	r := strings.ToLower(strings.TrimSpace(resp))
	return r == "y" || r == "yes", nil
}

func (p *TextPrompter) Prompt(q string) (string, error) {
	if _, err := fmt.Fprint(p.out, q); err != nil {
		return "", err
	}

	resp, err := p.in.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}
