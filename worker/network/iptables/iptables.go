package iptables

import (
	goiptables "github.com/coreos/go-iptables/iptables"
)

type iptables struct {
	goipt *goiptables.IPTables
}

func New() (*iptables, error) {
	g, err := goiptables.New()
	if err != nil {
		return nil, err
	}

	ipt := iptables{
		goipt: g,
	}

	return &ipt, nil
}

func (ipt *iptables) CreateChainOrFlushIfExists(table string, chain string) error {
	err := ipt.goipt.ClearChain(table, chain)
	return err
}

func (ipt *iptables) AppendRule(table string, chain string, rulespec ...string) error {
	err := ipt.goipt.Append(table, chain, rulespec...)
	return err
}

func (ipt *iptables) InsertRule(table string, chain string, pos int, rulespec ...string) error {
	err := ipt.goipt.Insert(table, chain, pos, rulespec...)
	return err
}

func (ipt *iptables) DeleteRule(table string, chain string, rulespec ...string) error {
	err := ipt.goipt.DeleteIfExists(table, chain, rulespec...)
	return err
}
