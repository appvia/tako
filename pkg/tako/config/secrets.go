/**
 * Copyright 2020 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

const (
	PartIdentifier = "id"
	PartValue      = "value"
)

// Sources,
// gitrob: https://github.com/michenriksen/gitrob/blob/master/core/signatures.go
// shhgit: https://github.com/eth0izzle/shhgit/blob/046cf0b704291aafba97d1feff19fe57e5e09152/config.yaml

var SecretMatchers = []map[string]string{
	{
		"part":        PartValue,
		"match":       ".pem",
		"description": "a Potential cryptographic private key",
	},
	{
		"part":        PartValue,
		"match":       ".log",
		"description": "a Log file",
		"comment":     "Log files can contain secret HTTP endpoints, session IDs, API keys and other goodies",
	},
	{
		"part":        PartValue,
		"match":       ".pkcs12",
		"description": "a Potential cryptographic key bundle",
	},
	{
		"part":        PartValue,
		"match":       ".p12",
		"description": "a Potential cryptographic key bundle",
	},
	{
		"part":        PartValue,
		"match":       ".pfx",
		"description": "a Potential cryptographic key bundle",
	},
	{
		"part":        PartValue,
		"match":       ".asc",
		"description": "a Potential cryptographic key bundle",
	},
	{
		"part":        PartValue,
		"match":       "otr.private_key",
		"description": "a Pidgin OTR private key",
	},
	{
		"part":        PartValue,
		"match":       ".ovpn",
		"description": "an OpenVPN client configuration file",
	},
	{
		"part":        PartValue,
		"match":       ".cscfg",
		"description": "an Azure service configuration schema file",
	},
	{
		"part":        PartValue,
		"match":       ".rdp",
		"description": "a Remote Desktop connection file",
	},
	{
		"part":        PartValue,
		"match":       ".mdf",
		"description": "a Microsoft SQL database file",
	},
	{
		"part":        PartValue,
		"match":       ".sdf",
		"description": "a Microsoft SQL server compact database file",
	},
	{
		"part":        PartValue,
		"match":       ".sqlite",
		"description": "a SQLite database file",
	},
	{
		"part":        PartValue,
		"match":       ".bek",
		"description": "a Microsoft BitLocker recovery key file",
	},
	{
		"part":        PartValue,
		"match":       ".tpm",
		"description": "a Microsoft BitLocker Trusted Platform Module password file",
	},
	{
		"part":        PartValue,
		"match":       ".fve",
		"description": "a Windows BitLocker full volume encrypted data file",
	},
	{
		"part":        PartValue,
		"match":       ".jks",
		"description": "a Java keystore file",
	},
	{
		"part":        PartValue,
		"match":       ".psafe3",
		"description": "a Password Safe database file",
	},
	{
		"part":        PartValue,
		"match":       "secret_token.rb",
		"description": "a Ruby On Rails secret token configuration file",
		"comment":     "If the Rails secret token is known, it can allow for remote code execution (http://www.exploit-db.com/exploits/27527/)",
	},
	{
		"part":        PartValue,
		"match":       "carrierwave.rb",
		"description": "a Carrierwave configuration file",
		"comment":     "Can contain credentials for cloud storage systems such as Amazon S3 and Google Storage",
	},
	{
		"part":        PartValue,
		"match":       "database.yml",
		"description": "a Potential Ruby On Rails database configuration file",
		"comment":     "Can contain database credentials",
	},
	{
		"part":        PartValue,
		"match":       "omniauth.rb",
		"description": "an OmniAuth configuration file",
		"comment":     "The OmniAuth configuration file can contain client application secrets",
	},
	{
		"part":        PartValue,
		"match":       "settings.py",
		"description": "a Django configuration file",
		"comment":     "Can contain database credentials, cloud storage system credentials, and other secrets",
	},
	{
		"part":        PartValue,
		"match":       ".agilekeychain",
		"description": "a 1Password password manager database file",
		"comment":     "Feed it to Hashcat and see if you're lucky",
	},
	{
		"part":        PartValue,
		"match":       ".keychain",
		"description": "a Apple Keychain database file",
	},
	{
		"part":        PartValue,
		"match":       ".pcap",
		"description": "a Network traffic capture file",
	},
	{
		"part":        PartValue,
		"match":       ".gnucash",
		"description": "a GnuCash database file",
	},
	{
		"part":        PartValue,
		"match":       "jenkins.plugins.publish_over_ssh.BapSshPublisherPlugin.xml",
		"description": "a Jenkins publish over SSH plugin file",
	},
	{
		"part":        PartValue,
		"match":       "credentials.xml",
		"description": "a Potential Jenkins credentials file",
	},
	{
		"part":        PartValue,
		"match":       ".kwallet",
		"description": "a KDE Wallet Manager database file",
	},
	{
		"part":        PartValue,
		"match":       "LocalSettings.php",
		"description": "a Potential MediaWiki configuration file",
	},
	{
		"part":        PartValue,
		"match":       ".tblk",
		"description": "a Tunnelblick VPN configuration file",
	},
	{
		"part":        PartValue,
		"match":       "a Favorites.plist",
		"description": "Sequel Pro MySQL database manager bookmark file",
	},
	{
		"part":        PartValue,
		"match":       "configuration.user.xpl",
		"description": "a Little Snitch firewall configuration file",
		"comment":     "Contains traffic rules for applications",
	},
	{
		"part":        PartValue,
		"match":       ".dayone",
		"description": "a Day One journal file",
		"comment":     "Now it's getting creepy...",
	},
	{
		"part":        PartValue,
		"match":       "journal.txt",
		"description": "a Potential jrnl journal file",
		"comment":     "Now it's getting creepy...",
	},
	{
		"part":        PartValue,
		"match":       "knife.rb",
		"description": "Chef Knife configuration file",
		"comment":     "Can contain references to Chef servers",
	},
	{
		"part":        PartValue,
		"match":       "proftpdpasswd",
		"description": "cPanel backup ProFTPd credentials file",
		"comment":     "Contains usernames and password hashes for FTP accounts",
	},
	{
		"part":        PartValue,
		"match":       "robomongo.json",
		"description": "Robomongo MongoDB manager configuration file",
		"comment":     "Can contain credentials for MongoDB databases",
	},
	{
		"part":        PartValue,
		"match":       "filezilla.xml",
		"description": "FileZilla FTP configuration file",
		"comment":     "Can contain credentials for FTP servers",
	},
	{
		"part":        PartValue,
		"match":       "recentservers.xml",
		"description": "FileZilla FTP recent servers file",
		"comment":     "Can contain credentials for FTP servers",
	},
	{
		"part":        PartValue,
		"match":       "ventrilo_srv.ini",
		"description": "Ventrilo server configuration file",
		"comment":     "Can contain passwords",
	},
	{
		"part":        PartValue,
		"match":       "terraform.tfvars",
		"description": "Terraform variable config file",
		"comment":     "Can contain credentials for terraform providers",
	},
	{
		"part":        PartValue,
		"match":       ".exports",
		"description": "Shell configuration file",
		"comment":     "a Shell configuration files can contain passwords, API keys, hostnames and other goodies",
	},
	{
		"part":        PartValue,
		"match":       ".functions",
		"description": "Shell configuration file",
		"comment":     "a Shell configuration files can contain passwords, API keys, hostnames and other goodies",
	},
	{
		"part":        PartValue,
		"match":       ".extra",
		"description": "Shell configuration file",
		"comment":     "a Shell configuration files can contain passwords, API keys, hostnames and other goodies",
	},
	{
		"part":        PartValue,
		"match":       `^.*_rsa$`,
		"description": "a Private SSH key",
	},
	{
		"part":        PartValue,
		"match":       `^.*_dsa$`,
		"description": "a Private SSH key",
	},
	{
		"part":        PartValue,
		"match":       `^.*_ed25519$`,
		"description": "a Private SSH key",
	},
	{
		"part":        PartValue,
		"match":       `^.*_ecdsa$`,
		"description": "a Private SSH key",
	},
	{
		"part":        PartValue,
		"match":       `\.?ssh/config$`,
		"description": "a SSH configuration file",
	},
	{
		"part":        PartValue,
		"match":       `^key(pair)?$`,
		"description": "a Potential cryptographic private key",
	},
	{
		"part":        PartValue,
		"match":       `^\.?(bash_|zsh_|sh_|z)?history$`,
		"description": "a Shell command history file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?mysql_history$`,
		"description": "a MySQL client command history file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?psql_history$`,
		"description": "a PostgreSQL client command history file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?pgpass$`,
		"description": "a PostgreSQL password file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?irb_history$`,
		"description": "a Ruby IRB console history file",
	},
	{
		"part":        PartValue,
		"match":       `\.?purple/accounts\.xml$`,
		"description": "a Pidgin chat client account configuration file",
	},
	{
		"part":        PartValue,
		"match":       `\.?xchat2?/servlist_?\.conf$`,
		"description": "a Hexchat/XChat IRC client server list configuration file",
	},
	{
		"part":        PartValue,
		"match":       `\.?irssi/config$`,
		"description": "an Irssi IRC client configuration file",
	},
	{
		"part":        PartValue,
		"match":       `\.?recon-ng/keys\.db$`,
		"description": "a Recon-ng web reconnaissance framework API key database",
	},
	{
		"part":        PartValue,
		"match":       `^\.?dbeaver-data-sources.xml$`,
		"description": "a DBeaver SQL database manager configuration file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?muttrc$`,
		"description": "a Mutt e-mail client configuration file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?s3cfg$`,
		"description": "an S3cmd configuration file",
	},
	{
		"part":        PartValue,
		"match":       `\.?aws/credentials$`,
		"description": "an AWS CLI credentials file",
	},
	{
		"part":        PartValue,
		"match":       `^sftp-config(\.json)?$`,
		"description": "a SFTP connection configuration file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?trc$`,
		"description": "a T command-line Twitter client configuration file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?gitrobrc$`,
		"description": "a Gitrob configuration file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?(bash|zsh|csh)rc$`,
		"description": "a Shell configuration file",
		"comment":     "Shell configuration files can contain passwords, API keys, hostnames and other goodies",
	},
	{
		"part":        PartValue,
		"match":       `^\.?(bash_|zsh_)?profile$`,
		"description": "a Shell profile configuration file",
		"comment":     "Shell configuration files can contain passwords, API keys, hostnames and other goodies",
	},
	{
		"part":        PartValue,
		"match":       `^\.?(bash_|zsh_)?aliases$`,
		"description": "a Shell command alias configuration file",
		"comment":     "Shell configuration files can contain passwords, API keys, hostnames and other goodies",
	},
	{
		"part":        PartValue,
		"match":       `config(\.inc)?\.php$`,
		"description": "a PHP configuration file",
	},
	{
		"part":        PartValue,
		"match":       `^key(store|ring)$`,
		"description": "a GNOME Keyring database file",
	},
	{
		"part":        PartValue,
		"match":       `^kdbx?$`,
		"description": "a KeePass password manager database file",
		"comment":     "Feed it to Hashcat and see if you're lucky",
	},
	{
		"part":        PartValue,
		"match":       `^sql(dump)?$`,
		"description": "a SQL dump file",
	},
	{
		"part":        PartValue,
		"match":       `^\.?htpasswd$`,
		"description": "an Apache htpasswd file",
	},
	{
		"part":        PartValue,
		"match":       `^(\.|_)?netrc$`,
		"description": "Configuration file for auto-login process",
		"comment":     "Can contain username and password",
	},
	{
		"part":        PartValue,
		"match":       `\.?gem/credentials$`,
		"description": "Rubygems credentials file",
		"comment":     "Can contain API key for a rubygems.org account",
	},
	{
		"part":        PartValue,
		"match":       `^\.?tugboat$`,
		"description": "a Tugboat DigitalOcean management tool configuration",
	},
	{
		"part":        PartValue,
		"match":       `doctl/config.yaml$`,
		"description": "DigitalOcean doctl command-line client configuration file",
		"comment":     "Contains DigitalOcean API key and other information",
	},
	{
		"part":        PartValue,
		"match":       `^\.?git-credentials$`,
		"description": "a git-credential-store helper credentials file",
	},
	{
		"part":        PartValue,
		"match":       `config/hub$`,
		"description": "GitHub Hub command-line client configuration file",
		"comment":     "Can contain GitHub API access token",
	},
	{
		"part":        PartValue,
		"match":       `^\.?gitconfig$`,
		"description": "a Git configuration file",
	},
	{
		"part":        PartValue,
		"match":       `\.?chef/(.*)\.pem$`,
		"description": "Chef private key",
		"comment":     "Can be used to authenticate against Chef servers",
	},
	{
		"part":        PartValue,
		"match":       `etc/shadow$`,
		"description": "Potential Linux shadow file",
		"comment":     "Contains hashed passwords for system users",
	},
	{
		"part":        PartValue,
		"match":       `etc/passwd$`,
		"description": "Potential Linux passwd file",
		"comment":     "Contains system user information",
	},
	{
		"part":        PartValue,
		"match":       `^\.?dockercfg$`,
		"description": "Docker configuration file",
		"comment":     "Can contain credentials for public or private Docker registries",
	},
	{
		"part":        PartValue,
		"match":       `^\.?npmrc$`,
		"description": "NPM configuration file",
		"comment":     "Can contain credentials for NPM registries",
	},
	{
		"part":        PartValue,
		"match":       `^\.?env$`,
		"description": "an Environment configuration file",
	},
	{
		"part":        PartValue,
		"match":       `(A3T[A-Z0-9]|AKIA|AGPA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`,
		"description": "an AWS Access Key ID Value",
	},
	{
		"part":        PartValue,
		"match":       "((\\\"|'|`)?((?i)aws)?_?((?i)access)_?((?i)key)?_?((?i)id)?(\\\"|'|`)?\\\\s{0,50}(:|=>|=)\\\\s{0,50}(\\\"|'|`)?(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}(\\\"|'|`)?)",
		"description": "an AWS Access Key ID",
	},
	{
		"part":        PartValue,
		"match":       "((\\\"|'|`)?((?i)aws)?_?((?i)account)_?((?i)id)?(\\\"|'|`)?\\\\s{0,50}(:|=>|=)\\\\s{0,50}(\\\"|'|`)?[0-9]{4}-?[0-9]{4}-?[0-9]{4}(\\\"|'|`)?)",
		"description": "an AWS Account ID",
	},
	{
		"part":        PartValue,
		"match":       "((\\\"|'|`)?((?i)aws)?_?((?i)secret)_?((?i)access)?_?((?i)key)?_?((?i)id)?(\\\"|'|`)?\\\\s{0,50}(:|=>|=)\\\\s{0,50}(\\\"|'|`)?[A-Za-z0-9/+=]{40}(\\\"|'|`)?)",
		"description": "an AWS Secret Access Key",
	},
	{
		"part":        PartValue,
		"match":       "((\\\"|'|`)?((?i)aws)?_?((?i)session)?_?((?i)token)?(\\\"|'|`)?\\\\s{0,50}(:|=>|=)\\\\s{0,50}(\\\"|'|`)?[A-Za-z0-9/+=]{16,}(\\\"|'|`)?)",
		"description": "an AWS Session Token",
	},
	{
		"part":        PartValue,
		"match":       "(?i)artifactory.{0,50}(\\\"|'|`)?[a-zA-Z0-9=]{112}(\\\"|'|`)?",
		"description": "an Artifactory API Key",
	},
	{
		"part":        PartValue,
		"match":       "(?i)codeclima.{0,50}(\\\"|'|`)?[0-9a-f]{64}(\\\"|'|`)?",
		"description": "a CodeClimateAPI Key",
	},
	{
		"part":        PartValue,
		"match":       `EAACEdEose0cBA[0-9A-Za-z]+`,
		"description": "a Facebook access token",
	},
	{
		"part":        PartValue,
		"match":       "((\\\"|'|`)?type(\\\"|'|`)?\\\\s{0,50}(:|=>|=)\\\\s{0,50}(\\\"|'|`)?service_account(\\\"|'|`)?,?)",
		"description": "a Google (GCM) Service account",
	},
	{
		"part":        PartValue,
		"match":       `(?:r|s)k_[live|test]_[0-9a-zA-Z]{24}`,
		"description": "a Stripe API key"},
	{
		"part":        PartValue,
		"match":       `[0-9]+-[0-9A-Za-z_]{32}\.apps\.googleusercontent\.com`,
		"description": "a Google OAuth Key"},
	{
		"part":        PartValue,
		"match":       `AIza[0-9A-Za-z\\-_]{35}`,
		"description": "a Google Cloud API Key",
	},
	{
		"part":        PartValue,
		"match":       `ya29\\.[0-9A-Za-z\\-_]+`,
		"description": "a Google OAuth Access Token",
	},
	{
		"part":        PartValue,
		"match":       `sk_[live|test]_[0-9a-z]{32}`,
		"description": "a Picatic API key",
	},
	{
		"part":        PartValue,
		"match":       `sq0atp-[0-9A-Za-z\-_]{22}`,
		"description": "a Square Access Token",
	},
	{
		"part":        PartValue,
		"match":       `sq0csp-[0-9A-Za-z\-_]{43}`,
		"description": "a Square OAuth Secret"},
	{
		"part":        PartValue,
		"match":       `access_token\$production\$[0-9a-z]{16}\$[0-9a-f]{32}`,
		"description": "a PayPal/Braintree Access Token",
	},
	{
		"part":        PartValue,
		"match":       `amzn\.mws\.[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`,
		"description": "an Amazon MWS Auth Token",
	},
	{
		"part":        PartValue,
		"match":       `SK[0-9a-fA-F]{32}`,
		"description": "a Twilio API Key",
	},
	{
		"part":        PartValue,
		"match":       `SG\.[0-9A-Za-z\-_]{22}\.[0-9A-Za-z\-_]{43}`,
		"description": "a SendGrid API Key",
	},
	{
		"part":        PartValue,
		"match":       `key-[0-9a-zA-Z]{32}`,
		"description": "a MailGun API Key",
	},
	{
		"part":        PartValue,
		"match":       `[0-9a-f]{32}-us[0-9]{12}`,
		"description": "a MailChimp API Key",
	},
	{
		"part":        PartValue,
		"match":       "sshpass -p.*['|\\\"]",
		"description": "an SSH Password",
	},
	{
		"part":        PartValue,
		"match":       `(https\\://outlook\\.office.com/webhook/[0-9a-f-]{36}\\@)`,
		"description": "an Outlook team",
	},
	{
		"part":        PartValue,
		"match":       "(?i)sauce.{0,50}(\\\"|'|`)?[0-9a-f-]{36}(\\\"|'|`)?",
		"description": "a Sauce Token",
	},
	{
		"part":        PartValue,
		"match":       `(xox[pboa]-[0-9]{12}-[0-9]{12}-[0-9]{12}-[a-z0-9]{32})`,
		"description": "a Slack Token",
	},
	{
		"part":        PartValue,
		"match":       `https://hooks.slack.com/services/T[a-zA-Z0-9_]{8}/B[a-zA-Z0-9_]{8}/[a-zA-Z0-9_]{24}`,
		"description": "a Slack Webhook",
	},
	{
		"part":        PartValue,
		"match":       "(?i)sonar.{0,50}(\\\"|'|`)?[0-9a-f]{40}(\\\"|'|`)?",
		"description": "a SonarQube Docs API Key",
	},
	{
		"part":        PartValue,
		"match":       "(?i)hockey.{0,50}(\\\"|'|`)?[0-9a-f]{32}(\\\"|'|`)?",
		"description": "a HockeyApp Key",
	},
	{
		"part":        PartValue,
		"match":       `([\w+]{1,24})(://)([^$<]{1})([^\s";]{1,}):([^$<]{1})([^\s";/]{1,})@[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,24}([^\s]+)`,
		"description": "has Username and password in URI",
	},
	{
		"part":        PartValue,
		"match":       `oy2[a-z0-9]{43}`,
		"description": "a NuGet API Key",
	},
	{
		"part":        PartValue,
		"match":       `hawk\.[0-9A-Za-z\-_]{20}\.[0-9A-Za-z\-_]{20}`,
		"description": "a StackHawk API Key",
	},
	{
		"part":        PartValue,
		"match":       `-----BEGIN (EC|RSA|DSA|OPENSSH|PGP) PRIVATE KEY`,
		"description": "Contains a private key",
	},
	{
		"part":        PartValue,
		"match":       `define(.{0,20})?(DB_CHARSET|NONCE_SALT|LOGGED_IN_SALT|AUTH_SALT|NONCE_KEY|DB_HOST|DB_PASSWORD|AUTH_KEY|SECURE_AUTH_KEY|LOGGED_IN_KEY|DB_NAME|DB_USER)(.{0,20})?[', '|"].{10,120}[', '|"]`,
		"description": "WP-Config",
	},
	{
		"part":        PartValue,
		"match":       `(?i)(aws_access_key_id|aws_secret_access_key)(.{0,20})?=.[0-9a-zA-Z\/+]{20,40}`,
		"description": "an AWS cred file info",
	},
	{
		"part":        PartValue,
		"match":       `(?i)(facebook|fb)(.{0,20})?(?-i)[', '\"][0-9a-f]{32}[', '\"]`,
		"description": "a Facebook Secret Key",
	},
	{
		"part":        PartValue,
		"match":       `(?i)(facebook|fb)(.{0,20})?[', '\"][0-9]{13,17}[', '\"]`,
		"description": "a Facebook Client ID",
	},
	{
		"part":        PartValue,
		"match":       `(?i)twitter(.{0,20})?[', '\"][0-9a-z]{35,44}[', '\"]`,
		"description": "a Twitter Secret Key",
	},
	{
		"part":        PartValue,
		"match":       `(?i)twitter(.{0,20})?[', '\"][0-9a-z]{18,25}[', '\"]`,
		"description": "a Twitter Client ID",
	},
	{
		"part":        PartValue,
		"match":       `(?i)github(.{0,20})?(?-i)[', '\"][0-9a-zA-Z]{35,40}[', '\"]`,
		"description": "a Github Key",
	},
	{
		"part":        PartValue,
		"match":       `(?i)heroku(.{0,20})?[', '"][0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}[', '"]`,
		"description": "a Heroku API key",
	},
	{
		"part":        PartValue,
		"match":       `(?i)linkedin(.{0,20})?(?-i)[', '\"][0-9a-z]{12}[', '\"]`,
		"description": "a Linkedin Client ID",
	},
	{
		"part":        PartValue,
		"match":       `(?i)linkedin(.{0,20})?[', '\"][0-9a-z]{16}[', '\"]`,
		"description": "a LinkedIn Secret Key",
	},
	{
		"part":        PartIdentifier,
		"match":       `(?i)credential`,
		"description": "Contains word: credential",
	},
	{
		"part":        PartIdentifier,
		"match":       `(?i)user`,
		"description": "Contains word: user",
	},
	{
		"part":        PartIdentifier,
		"match":       `(?i).*password.*`,
		"description": "Contains word: password",
	},
	{
		"part":        PartIdentifier,
		"match":       `(?i)host`,
		"description": "Contains word: host",
	},
	{
		"part":        PartIdentifier,
		"match":       `(?i)(access)?.*_?.*(key)?.*_?.*id`,
		"description": "Contains words: (access or key) and id",
	},
	{
		"part":        PartIdentifier,
		"match":       `(?i)(secret)?.*_?.*(access)?.*_?.*key`,
		"description": "Contains words: (secret or access) and key",
	},
}
