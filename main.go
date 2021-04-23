package main

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

const slapd_conf = `
#modulepath  /usr/lib/openldap
#moduleload  back_mdb.so
#moduleload  pw-sha2.so
#moduleload  syncprov.so

include  {{ .OPENLDAP_CONF_DIR }}/schema/core.schema
include  {{ .OPENLDAP_CONF_DIR }}/schema/cosine.schema
include  {{ .OPENLDAP_CONF_DIR }}/schema/inetorgperson.schema
include  {{ .OPENLDAP_CONF_DIR }}/schema/nis.schema

pidfile   {{ .OPENLDAP_RUN_DIR }}/slapd.pid
argsfile  {{ .OPENLDAP_RUN_DIR }}/slapd.args

database   mdb
maxsize    {{ .OPENLDAP_MDB_MAXSIZE }}
suffix     "{{ .OPENLDAP_BASE_DN }}"
rootdn     "{{ .OPENLDAP_ROOT_DN }}"
rootpw     "{{ .OPENLDAP_ROOT_PW }}"
directory  {{ .OPENLDAP_DATA_DIR }}

index  objectClass                      eq
index  entryCSN                         eq
index  entryUUID                        eq
index  uidNumber,gidNumber              eq,pres
index  cn,sn,gn,uid,mail,email          eq,pres
index  createTimestamp,modifyTimestamp  eq,pres
`

const slapd_ldif = `
dn: {{ .OPENLDAP_BASE_DN }}
objectClass: top
objectClass: dcObject
objectClass: organization
o: example
dc: example

dn: ou=Computers,{{ .OPENLDAP_BASE_DN }}
objectClass: organizationalUnit
ou: Computers

dn: ou=Groups,{{ .OPENLDAP_BASE_DN }}
objectClass: organizationalUnit
ou: Groups

dn: ou=Roles,{{ .OPENLDAP_BASE_DN }}
objectClass: organizationalUnit
ou: Roles

dn: ou=Users,{{ .OPENLDAP_BASE_DN }}
objectClass: organizationalUnit
ou: Users
`

type Config struct {
	OPENLDAP_CONF_DIR    string
	OPENLDAP_DATA_DIR    string
	OPENLDAP_RUN_DIR     string
	OPENLDAP_MDB_MAXSIZE string
	OPENLDAP_BASE_DN     string
	OPENLDAP_ROOT_DN     string
	OPENLDAP_ROOT_PW     string
	OPENLDAP_REPLICA_DN  string
	OPENLDAP_REPLICA_PW  string
	OPENLDAP_SERVER_LIST string
}

func main() {
	c := &Config{
		OPENLDAP_CONF_DIR:    "/etc/openldap",
		OPENLDAP_DATA_DIR:    "/var/lib/openldap",
		OPENLDAP_RUN_DIR:     "/run/openldap",
		OPENLDAP_MDB_MAXSIZE: "1073741824",
		OPENLDAP_BASE_DN:     "dc=example,dc=com",
		OPENLDAP_ROOT_DN:     "cn=admin,dc=example,dc=com",
		OPENLDAP_ROOT_PW:     "admin",
		OPENLDAP_REPLICA_DN:  "cn=admin,dc=example,dc=com",
		OPENLDAP_REPLICA_PW:  "admin",
		OPENLDAP_SERVER_LIST: "localhost",
	}

	if _, err := os.Stat("/etc/openldap/slapd.d"); os.IsNotExist(err) {
		if err := os.MkdirAll("/etc/openldap/slapd.d", 0755); err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat("/var/lib/openldap"); os.IsNotExist(err) {
		if err := os.MkdirAll("/var/lib/openldap", 0755); err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat("/run/openldap"); os.IsNotExist(err) {
		if err := os.MkdirAll("/run/openldap", 0755); err != nil {
			panic(err)
		}
	}

	configure(c)
	initialize(c)

	flag.Parse()

	if flag.Arg(0) != "slapd" {
		shell()
	} else {
		slapd("add")
		slapd("index")
		slapd("test")
		slapd("")
	}
}

func configure(c *Config) {
	f, err := os.Create("/etc/openldap/slapd.conf")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	t, err := template.New("slapd_conf").Parse(strings.TrimSpace(slapd_conf))
	if err != nil {
		panic(err)
	}

	if err = t.Execute(f, c); err != nil {
		panic(err)
	}
}

func initialize(c *Config) {
	f, err := os.Create("/etc/openldap/slapd.ldif")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	t, err := template.New("slapd_ldif").Parse(strings.TrimSpace(slapd_ldif))
	if err != nil {
		panic(err)
	}

	if err = t.Execute(f, c); err != nil {
		panic(err)
	}
}

func slapd(command string) {
	args := []string{}

	switch command {
	case "add":
		args = append(args, []string{
			"-T", command,
			"-d", "1",
			"-f", "/etc/openldap/slapd.conf",
			"-F", "/etc/openldap/slapd.d",
			"-l", "/etc/openldap/slapd.ldif",
		}...)
	case "index", "test":
		args = append(args, []string{
			"-T", command,
			"-d", "1",
			"-f", "/etc/openldap/slapd.conf",
			"-F", "/etc/openldap/slapd.d",
		}...)
	case "":
		args = append(args, []string{
			"-d", "1",
			"-F", "/etc/openldap/slapd.d",
			"-h", "ldap:/// ldapi:///",
		}...)
	default:
		panic("unknown slapd command.")
	}

	cmd := exec.Command("/usr/libexec/slapd", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func shell() {
	if _, err := os.Stat("/busybox/sh"); os.IsNotExist(err) {
		panic(err)
	}

	cmd := exec.Command("/busybox/sh")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
