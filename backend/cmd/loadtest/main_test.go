package main

import "testing"

func TestAuthenticate_AccessTokenSkipsLogin(t *testing.T) {
	c := &gqlClient{}
	if err := c.authenticate(" jet ", "", ""); err != nil {
		t.Fatal(err)
	}
	if c.token != "jet" {
		t.Fatalf("token = %q", c.token)
	}
}

func TestAuthenticate_RequiresTokenPair(t *testing.T) {
	c := &gqlClient{}
	if err := c.authenticate("", "gecici", ""); err == nil {
		t.Fatal("mfa'sız geçici jeton kabul edildi")
	}
	if err := c.authenticate("", "", "123456"); err == nil {
		t.Fatal("jetonsuz mfa kabul edildi")
	}
}
