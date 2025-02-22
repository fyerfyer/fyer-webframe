package orm

import "database/sql"

type Result struct {
	res sql.Result
	err error
}

type ResultHandler interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

func (r Result) LastInsertId() (int64, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.res.LastInsertId()
}

func (r Result) RowsAffected() (int64, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.res.RowsAffected()
}
