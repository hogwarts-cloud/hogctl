package validator

import (
	"errors"
	"fmt"
	"net/mail"

	"github.com/hogwarts-cloud/hogctl/internal/models"
	"github.com/samber/lo"
	"golang.org/x/crypto/ssh"
)

var (
	ErrEmptyInstancesList           = errors.New("empty instances list")
	ErrEmptyInstanceName            = errors.New("instance name is empty")
	ErrInstanceNameTooBig           = errors.New("instance name is too big (max 256 characters)")
	ErrInvalidFlavor                = errors.New("invalid flavor")
	ErrNonPositiveDiskSize          = errors.New("disk size must be positive")
	ErrEmptyUserName                = errors.New("user name is empty")
	ErrUserNameTooBig               = errors.New("user name is too big (max 256 characters)")
	ErrInvalidEmail                 = errors.New("invalid email")
	ErrInvalidPublicKey             = errors.New("invalid public key")
	ErrFoundDuplicatedInstanceNames = errors.New("found duplicated instance names")
)

type Validator struct {
	flavors []models.Flavor
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

	if len(instance.Name) > 256 {
		return ErrInstanceNameTooBig
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

	if len(instance.User.Name) > 256 {
		return ErrUserNameTooBig
	}

	if _, err := mail.ParseAddress(instance.User.Email); err != nil {
		return ErrInvalidEmail
	}

	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(instance.User.PublicKey)); err != nil {
		return ErrInvalidPublicKey
	}

	return nil
}

func New(flavors []models.Flavor) *Validator {
	return &Validator{flavors: flavors}
}
