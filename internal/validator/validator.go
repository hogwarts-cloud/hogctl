package validator

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/hogwarts-cloud/hogctl/internal/models"
	"github.com/samber/lo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/idna"
)

var (
	ErrEmptyInstancesList           = errors.New("empty instances list")
	ErrEmptyInstanceName            = errors.New("empty instance name")
	ErrInvalidInstanceName          = errors.New("invalid instance name")
	ErrInstanceNameTooBig           = errors.New("instance name too big")
	ErrInvalidFlavor                = errors.New("invalid flavor")
	ErrNonPositiveDiskSize          = errors.New("non positive disk size")
	ErrEmptyUserName                = errors.New("empty user name")
	ErrUserNameTooBig               = errors.New("user name too big")
	ErrInvalidEmail                 = errors.New("invalid email")
	ErrInvalidPublicKey             = errors.New("invalid public key")
	ErrExpiredInstance              = errors.New("expired instance")
	ErrFoundDuplicatedInstanceNames = errors.New("found duplicated instance names")
)

type Validator struct {
	flavors []models.Flavor
	domain  string
}

func (v *Validator) Validate(instances []models.Instance) error {
	if len(instances) == 0 {
		return ErrEmptyInstancesList
	}

	set := make(map[string]struct{}, len(instances))

	for _, instance := range instances {
		set[instance.Name] = struct{}{}
		if err := v.validateInstance(instance); err != nil {
			return fmt.Errorf("failed to validate instance '%s': %w", instance.Name, err)
		}
	}

	if len(set) < len(instances) {
		return ErrFoundDuplicatedInstanceNames
	}

	return nil
}

func (v *Validator) validateInstance(instance models.Instance) error {
	if len(instance.Name) == 0 {
		return ErrEmptyInstanceName
	}

	if len(instance.Name) > 64 {
		return ErrInstanceNameTooBig
	}

	if _, err := idna.Lookup.ToASCII(fmt.Sprintf("%s.%s", instance.Name, v.domain)); err != nil ||
		strings.Contains(instance.Name, ".") {
		return ErrInvalidInstanceName
	}

	if !lo.ContainsBy(v.flavors, func(flavor models.Flavor) bool {
		return flavor.Name == instance.Resources.Flavor
	}) {
		return ErrInvalidFlavor
	}

	if instance.Resources.Disk <= 0 {
		return ErrNonPositiveDiskSize
	}

	if len(instance.User.Name) == 0 {
		return ErrEmptyUserName
	}

	if len(instance.User.Name) > 64 {
		return ErrUserNameTooBig
	}

	if _, err := mail.ParseAddress(instance.User.Email); err != nil {
		return ErrInvalidEmail
	}

	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(instance.User.PublicKey)); err != nil {
		return ErrInvalidPublicKey
	}

	if instance.ExpirationDate.Before(time.Now()) {
		return ErrExpiredInstance
	}

	return nil
}

func New(flavors []models.Flavor, domain string) *Validator {
	return &Validator{
		flavors: flavors,
		domain:  domain,
	}
}
