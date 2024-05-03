package validator

import (
	"errors"
	"fmt"
	"net/mail"
	"slices"

	"github.com/danilkaz/hogwarts-cloud/hogctl/internal/models"
	"golang.org/x/crypto/ssh"
)

var (
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

func Validate(instances []*models.Instance) error {
	set := make(map[string]struct{}, len(instances))

	for _, instance := range instances {
		set[instance.Name] = struct{}{}
		if err := validateInstance(instance); err != nil {
			return fmt.Errorf("failed to validate instance '%s': %w", instance.Name, err)
		}
	}

	if len(set) < len(instances) {
		return ErrFoundDuplicatedInstanceNames
	}

	return nil
}

func validateInstance(instance *models.Instance) error {
	if len(instance.Name) == 0 {
		return ErrEmptyInstanceName
	}

	if len(instance.Name) > 256 {
		return ErrInstanceNameTooBig
	}

	if !slices.Contains(models.AvailableFlavors, instance.Resources.Flavor) {
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
