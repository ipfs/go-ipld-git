package ipldgit

import (
	"bytes"
	"fmt"
)

func parsePersonInfo(line []byte) (PersonInfo, error) {
	parts := bytes.Split(line, []byte{' '})
	if len(parts) < 3 {
		return nil, fmt.Errorf("incorrectly formatted person info line: %q", line)
	}

	//TODO: just use regex?
	//skip prefix
	at := 1

	var pi _PersonInfo
	var name string

	for {
		if at == len(parts) {
			return nil, fmt.Errorf("invalid personInfo: %q", line)
		}
		part := parts[at]
		if len(part) != 0 {
			if part[0] == '<' {
				break
			}
			name += string(part) + " "
		} else if len(name) > 0 {
			name += " "
		}
		at++
	}
	if len(name) != 0 {
		pi.Name = _String{name[:len(name)-1]}
	}

	var email string
	for {
		if at == len(parts) {
			return nil, fmt.Errorf("invalid personInfo: %q", line)
		}
		part := parts[at]
		if part[0] == '<' {
			part = part[1:]
		}

		at++
		if part[len(part)-1] == '>' {
			email += string(part[:len(part)-1])
			break
		}
		email += string(part) + " "
	}
	pi.Email = _String{email}

	if at == len(parts) {
		return &pi, nil
	}
	pi.Date = _String{string(parts[at])}

	at++
	if at == len(parts) {
		return &pi, nil
	}
	pi.Timezone = _String{string(parts[at])}
	return &pi, nil
}

func (p _PersonInfo) GitString() string {
	f := "%s <%s>"
	arg := []interface{}{p.Name.x, p.Email.x}
	if p.Date.x != "" {
		f = f + " %s"
		arg = append(arg, p.Date.x)
	}

	if p.Timezone.x != "" {
		f = f + " %s"
		arg = append(arg, p.Timezone.x)
	}
	return fmt.Sprintf(f, arg...)
}
