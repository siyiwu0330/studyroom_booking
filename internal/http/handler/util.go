package handler

import "strconv"

func parseID(s string, out *int64) error {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*out = v
	return nil
}
