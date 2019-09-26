package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"

	"github.com/phoreproject/synapse/primitives"

	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/phoreproject/synapse/beacon/module"

	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/chainhash"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/validator"
	"github.com/prysmaticlabs/go-ssz"
)

var zeroHash = chainhash.Hash{}

func generateKeyFile(validatorsToGenerate string, rootkey string, f io.Writer) {
	validatorsStrings := strings.Split(validatorsToGenerate, ",")
	var validatorIndices []uint32

	for _, s := range validatorsStrings {
		if !strings.ContainsRune(s, '-') {
			i, err := strconv.Atoi(s)
			if err != nil {
				panic("invalid validators parameter")
			}
			validatorIndices = append(validatorIndices, uint32(i))
		} else {
			parts := strings.SplitN(s, "-", 2)
			if len(parts) != 2 {
				panic("invalid validators parameter")
			}
			first, err := strconv.Atoi(parts[0])
			if err != nil {
				panic("invalid validators parameter")
			}
			second, err := strconv.Atoi(parts[1])
			if err != nil {
				panic("invalid validators parameter")
			}
			for i := first; i <= second; i++ {
				validatorIndices = append(validatorIndices, uint32(i))
			}
		}
	}

	validators := make([]module.InitialValidatorInformation, len(validatorIndices))

	for i, v := range validatorIndices {
		key, _ := bls.RandSecretKey(validator.GetReaderForID(rootkey, v))

		pub := key.DerivePublicKey()

		pubSer := pub.Serialize()

		h, err := ssz.HashTreeRoot(pubSer)
		if err != nil {
			panic(err)
		}

		sig, err := bls.Sign(key, h[:], bls.DomainDeposit)
		if err != nil {
			panic(err)
		}

		sigSer := sig.Serialize()

		iv := module.InitialValidatorInformation{
			PubKey:                fmt.Sprintf("%x", pubSer),
			ProofOfPossession:     fmt.Sprintf("%x", sigSer),
			WithdrawalShard:       0,
			WithdrawalCredentials: fmt.Sprintf("%x", zeroHash[:]),
			DepositSize:           config.MainNetConfig.MaxDeposit,
			ID:                    v,
		}

		validators[i] = iv
	}

	vList := module.InitialValidatorList{
		NumValidators: len(validatorIndices),
		Validators:    validators,
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err := enc.Encode(vList)
	if err != nil {
		panic(err)
	}
}

// ByID sorts validator information by ID.
type ByID []module.InitialValidatorInformation

func (b ByID) Len() int { return len(b) }

func (b ByID) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b ByID) Less(i, j int) bool {
	return b[i].ID < b[j].ID
}

func main() {
	generate := flag.Bool("generate", false, "generate validator key files")
	rootkey := flag.String("rootkey", "", "this key derives all other keys")
	validators := flag.String("validators", "", "validator assignment")
	outfile := flag.String("outfile", "validators.json", "file with all of the public keys of validators")

	combine := flag.Bool("combine", false, "combine validator key files")
	combineFiles := flag.String("input", "", "space separated list of files to combine")
	flag.Parse()

	if !*combine && !*generate {
		panic("Expected either -combine or -generate")
	}

	var filesToCombine []string

	if *combine {
		filesToCombine = strings.Split(*combineFiles, " ")

		if len(filesToCombine) == 0 {
			panic("expected files to combine")
		}
	}

	if *generate {
		f, err := os.Create(*outfile)
		if err != nil {
			panic(err)
		}

		generateKeyFile(*validators, *rootkey, f)

		err = f.Close()
		if err != nil {
			panic(err)
		}
	} else if *combine {
		validatorList := make(map[uint32]primitives.InitialValidatorEntry)

		for _, fileToCombine := range filesToCombine {
			// we should load the keys from the validator keystore
			f, err := os.Open(fileToCombine)
			if err != nil {
				panic(err)
			}

			decoder := json.NewDecoder(f)

			var ivList module.InitialValidatorList

			err = decoder.Decode(&ivList)
			if err == nil {
				for _, v := range ivList.Validators {
					pubKeyBytes, err := hex.DecodeString(v.PubKey)
					if err != nil {
						panic(err)
					}
					var pubKey [96]byte
					copy(pubKey[:], pubKeyBytes)

					sigBytes, err := hex.DecodeString(v.ProofOfPossession)
					if err != nil {
						panic(err)
					}
					var signature [48]byte
					copy(signature[:], sigBytes)

					withdrawalCredentialsBytes, err := hex.DecodeString(v.WithdrawalCredentials)
					if err != nil {
						panic(err)
					}
					var withdrawalCredentials [32]byte
					copy(withdrawalCredentials[:], withdrawalCredentialsBytes)

					validatorList[v.ID] = primitives.InitialValidatorEntry{
						PubKey:                pubKey,
						ProofOfPossession:     signature,
						WithdrawalCredentials: withdrawalCredentials,
						WithdrawalShard:       v.WithdrawalShard,
						DepositSize:           v.DepositSize,
					}
				}

				continue
			}

			if _, err := f.Seek(0, 0); err != nil {
				panic(err)
			}

			var lengthBytes [4]byte

			_, err = f.Read(lengthBytes[:])
			if err != nil {
				panic(err)
			}

			length := binary.BigEndian.Uint32(lengthBytes[:])

			for i := uint32(0); i < length; i++ {
				var validatorIDBytes [4]byte
				n, err := f.Read(validatorIDBytes[:])
				if err != nil {
					panic(err)
				}
				if n != 4 {
					panic("unexpected end of pubkey file")
				}

				validatorID := binary.BigEndian.Uint32(validatorIDBytes[:])

				var iv primitives.InitialValidatorEntry
				err = binary.Read(f, binary.BigEndian, &iv)
				if err != nil {
					panic(err)
				}

				validatorList[validatorID] = iv
			}
			f.Close()
		}

		validators := make([]module.InitialValidatorInformation, len(validatorList))

		i := 0

		for index, entry := range validatorList {
			validators[i] = module.InitialValidatorInformation{
				PubKey:                fmt.Sprintf("%x", entry.PubKey),
				ProofOfPossession:     fmt.Sprintf("%x", entry.ProofOfPossession),
				WithdrawalShard:       entry.WithdrawalShard,
				WithdrawalCredentials: fmt.Sprintf("%x", entry.WithdrawalCredentials[:]),
				DepositSize:           entry.DepositSize,
				ID:                    index,
			}
			i++
		}

		f, err := os.Create(*outfile)
		if err != nil {
			panic(err)
		}

		sort.Sort(ByID(validators))

		vList := module.InitialValidatorList{
			NumValidators: len(validators),
			Validators:    validators,
		}

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		err = enc.Encode(vList)
		if err != nil {
			panic(err)
		}

		err = f.Close()
		if err != nil {
			panic(err)
		}
	}
}
