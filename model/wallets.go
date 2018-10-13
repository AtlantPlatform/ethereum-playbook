package model

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Wallets map[string]*WalletSpec

func (wallets Wallets) Validate(ctx AppContext, spec *Spec) bool {
	for name, wallet := range wallets {
		if !wallet.Validate(ctx, name) {
			return false
		}
	}
	return true
}

func (wallets Wallets) NameForAddress(address common.Address) string {
	ref := strings.ToLower(address.Hex())
	for name, wallet := range wallets {
		if wallet.Address == ref {
			return name
		}
	}
	return ""
}

func (wallets Wallets) WalletSpec(name string) (*WalletSpec, bool) {
	w, ok := wallets[name]
	return w, ok
}

type WalletSpec struct {
	Address  string `yaml:"address"`
	PrivKey  string `yaml:"privkey"`
	Password string `yaml:"password"`
	KeyStore string `yaml:"keystore"`
	KeyFile  string `yaml:"keyfile"`

	privKey *ecdsa.PrivateKey `yaml:"-"`
}

func (spec *WalletSpec) Validate(ctx AppContext, name string) bool {
	validateLog := log.WithFields(log.Fields{
		"section": "Wallets",
		"wallet":  name,
	})
	if len(spec.Address) > 0 {
		if spec.Address != "0x0" && !common.IsHexAddress(spec.Address) {
			validateLog.Errorln("address is not valid (must be hex string starting from 0x)")
			return false
		}
	}
	account := common.HexToAddress(spec.Address)
	if len(spec.PrivKey) > 0 {
		if len(spec.Password) > 0 {
			validateLog.Warningln("private key is being loaded from string, but password is provided")
		}
		if len(spec.KeyFile) > 0 {
			validateLog.Warningln("private key is being loaded from string, but keyfile is provided")
		}
		// priv key being loaded UNPROTECTED, no need to provide password or disk access
		pk, err := crypto.HexToECDSA(spec.PrivKey)
		if err != nil {
			validateLog.WithError(err).Errorln("failed to unpack priv key from hex bytes (must be ...)")
			return false
		}
		accountFromPub := crypto.PubkeyToAddress(pk.PublicKey)
		if len(spec.Address) == 0 || spec.Address == "0x0" {
			spec.Address = strings.ToLower(accountFromPub.Hex())
			validateLog.WithFields(log.Fields{
				"address": spec.Address,
			}).Infoln("loaded address from privkey")
		} else if !bytes.Equal(accountFromPub.Bytes(), account.Bytes()) {
			validateLog.WithFields(log.Fields{
				"address": spec.Address,
			}).Errorln("address loaded from privkey differs from specified address")
			return false
		}
		spec.privKey = pk
		spec.PrivKey = ""
		// at this point private key is loaded and cached
		// we are ready to use the wallet.
		return true
	}
	if len(spec.KeyFile) > 0 {
		if len(spec.Password) == 0 {
			validateLog.Errorln("no password is provided for the account keyfile")
			return false
		}
		if strings.HasPrefix(spec.KeyFile, "keystore://") {
			if len(spec.KeyStore) > 0 {
				validateLog.Warningln(
					"replacing keystore path with keyfile dir, detected keystore:// prefix")
			}
			spec.KeyFile = strings.TrimPrefix(spec.KeyFile, "keystore://")
			spec.KeyStore = filepath.Dir(filepath.FromSlash(spec.KeyFile))
			spec.KeyFile = filepath.Base(spec.KeyFile)
			// at this point the original path was:
			// "keystore://" + filepath.Join(spec.KeyStore, spec.KeyFile)
		} else {
			storeAbs := filepath.IsAbs(spec.KeyStore)
			fileAbs := filepath.IsAbs(spec.KeyFile)
			if storeAbs && fileAbs {
				validateLog.Warningln(
					"removing keystore path, since keyfile path was absolute")
				spec.KeyStore = ""
			}
			if storeAbs {
				spec.KeyStore = filepath.FromSlash(spec.KeyStore)
			} else if fileAbs {
				spec.KeyFile = filepath.FromSlash(spec.KeyFile)
			}
		}
		keyFilePath := filepath.Join(spec.KeyStore, spec.KeyFile)
		keyFileLog := validateLog.WithField("keyfile", keyFilePath)
		if !isFile(keyFilePath) {
			keyFileLog.Errorln("file specified in keyfile is not found or cannot be read")
			return false
		} else if keyFile, err := loadKeyFile(keyFilePath); err != nil {
			keyFileLog.WithError(err).Errorln("file specified in keyfile has wrong format")
			return false
		} else {
			accountFromKeyfile := keyFile.HexToAddress()
			if len(spec.Address) == 0 || spec.Address == "0x0" {
				account = accountFromKeyfile
				spec.Address = strings.ToLower(accountFromKeyfile.Hex())
				validateLog.WithFields(log.Fields{
					"address": spec.Address,
				}).Infoln("loaded address from keyfile")
			} else if !bytes.Equal(accountFromKeyfile.Bytes(), account.Bytes()) {
				keyFileLog.WithFields(log.Fields{
					"address":        spec.Address,
					"keyfileAddress": strings.ToLower(accountFromKeyfile.Hex()),
				}).Errorln("address loaded from keyfile differs from specified address")
				return false
			}
		}
		ctx.KeyCache().SetPath(account, keyFilePath)
		pk, ok := ctx.KeyCache().PrivateKey(account, spec.Password)
		if !ok {
			keyFileLog.Errorln("unable to load private key from keyfile")
			ctx.KeyCache().UnsetPath(account, keyFilePath)
			return false
		}
		accountFromPub := crypto.PubkeyToAddress(pk.PublicKey)
		if !bytes.Equal(accountFromPub.Bytes(), account.Bytes()) {
			keyFileLog.WithFields(log.Fields{
				"address":        spec.Address,
				"keyfileAddress": strings.ToLower(accountFromPub.Hex()),
			}).Errorln("address loaded from keyfile differs from specified address")
			ctx.KeyCache().UnsetPath(account, keyFilePath)
			return false
		}
		// at this point private key is loaded and cached
		// we are ready to use the wallet.
		return true
	}
	if len(spec.KeyStore) == 0 {
		validateLog.Errorln("no privkey, keyfile or keystore prefix specified")
		return false
	} else if len(spec.Address) == 0 {
		validateLog.Errorln("no account is specified to search the keyfile in keystore prefix")
		return false
	} else if len(spec.Password) == 0 {
		validateLog.Errorln("no password is provided for the account keyfile")
		return false
	}
	var accountKeyfile *keyFile
	if err := forEachKeyFile(spec.KeyStore, func(keyfile *keyFile) error {
		if bytes.Equal(keyfile.HexToAddress().Bytes(), account.Bytes()) {
			accountKeyfile = keyfile
			return errStopRange
		}
		return nil
	}); err != nil {
		validateLog.WithError(err).Errorln("failed to search keyfile in keystore")
		return false
	}
	if accountKeyfile == nil {
		validateLog.WithFields(log.Fields{
			"address": spec.Address,
		}).Errorln("failed to locate private key")
		return false
	}
	keyFileLog := validateLog.WithField("keyfile", accountKeyfile.Path)
	ctx.KeyCache().SetPath(account, accountKeyfile.Path)
	pk, ok := ctx.KeyCache().PrivateKey(account, spec.Password)
	if !ok {
		keyFileLog.Errorln("unable to load private key from keyfile")
		ctx.KeyCache().UnsetPath(account, accountKeyfile.Path)
		return false
	}
	accountFromPub := crypto.PubkeyToAddress(pk.PublicKey)
	if !bytes.Equal(accountFromPub.Bytes(), account.Bytes()) {
		keyFileLog.WithFields(log.Fields{
			"address":        spec.Address,
			"keyfileAddress": strings.ToLower(accountFromPub.Hex()),
		}).Errorln("address loaded from keyfile differs from specified address")
		ctx.KeyCache().UnsetPath(account, accountKeyfile.Path)
		return false
	}
	validateLog.WithFields(log.Fields{
		"address": spec.Address,
	}).Infoln("located keyfile by address")
	// at this point private key is loaded and cached
	// we are ready to use the wallet.
	return true
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	} else if info.IsDir() {
		return false
	}
	return true
}

const (
	WalletSpecAddressField  FieldName = "address"
	WalletSpecPasswordField FieldName = "password"
	WalletSpecKeyStoreField FieldName = "keystore"
	WalletSpecKeyFileField  FieldName = "keyfile"
)

func (spec *WalletSpec) HasField(name FieldName) bool {
	switch name {
	case WalletSpecAddressField,
		WalletSpecPasswordField,
		WalletSpecKeyStoreField,
		WalletSpecKeyFileField:
		return true
	default:
		return false
	}
}

func (spec *WalletSpec) FieldValue(name FieldName) string {
	switch name {
	case WalletSpecAddressField:
		return spec.Address
	case WalletSpecPasswordField:
		return spec.Password
	case WalletSpecKeyStoreField:
		return spec.KeyStore
	case WalletSpecKeyFileField:
		return spec.KeyFile
	default:
		panic("value of non-existing field")
	}
}

func listAccounts(paths ...string) (accounts []common.Address) {
	for _, keystorePath := range paths {
		if err := forEachKeyFile(keystorePath, func(keyfile *keyFile) error {
			accounts = append(accounts, keyfile.HexToAddress())
			return nil
		}); err != nil {
			continue
		}
	}
	return accounts
}

var errStopRange = errors.New("stop")

func forEachKeyFile(keystorePath string, fn func(keyfile *keyFile) error) error {
	if err := filepath.Walk(keystorePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if path == keystorePath {
			return nil
		} else if info.IsDir() {
			return filepath.SkipDir
		}
		keyfile, err := loadKeyFile(path)
		if err != nil {
			return err
		}
		return fn(keyfile)
	}); err == errStopRange {
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

func loadKeyFile(path string) (*keyFile, error) {
	var keyfile *keyFile
	if data, err := ioutil.ReadFile(path); err != nil {
		return nil, err
	} else if err = json.Unmarshal(data, &keyfile); err != nil {
		return nil, err
	}
	if len(keyfile.Address) == 0 {
		err := fmt.Errorf("failed to load address from %s", path)
		return nil, err
	} else if !common.IsHexAddress(keyfile.Address) {
		err := fmt.Errorf("wrong (not hex) address from %s", path)
		return nil, err
	}
	keyfile.Path = path
	return keyfile, nil
}

type keyFile struct {
	Address string `json:"address"`
	ID      string `json:"id"`
	Version int    `json:"version"`
	Path    string `json:"-"`
}

func (keyfile *keyFile) HexToAddress() common.Address {
	return common.HexToAddress(keyfile.Address)
}
