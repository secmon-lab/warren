package bigquery

func (x *Action) ByteLimit() int64 {
	return x.byteLimit
}

func (x *Action) LimitRows() int64 {
	return x.cfg.Limit.Rows
}

func (x *Action) Tables() []TableConfig {
	return x.cfg.Tables
}
