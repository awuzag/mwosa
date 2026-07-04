package core

import (
	"reflect"
	"sort"

	"github.com/samber/oops"
)

type RoleProfile struct {
	Role          Role
	Markets       []Market
	SecurityTypes []SecurityType
	Group         GroupID
	Operations    []OperationID
	AuthScope     CredentialScope
	Freshness     Freshness
	Compatibility Compatibility
	RequiresAuth  bool
	Priority      int
	Limitations   []string
	FetchMode     string
}

type RoleRegistration struct {
	Profile RoleProfile
	Impl    any
}

type RoleProvider interface {
	RoleRegistration() RoleRegistration
}

type RoleRegistrationProvider interface {
	RoleRegistrations() []RoleRegistration
}

type GroupRoleProvider interface {
	RoleRegistrationProvider
	ProviderGroup() GroupID
}

type RoleEntry struct {
	Provider Identity
	Profile  RoleProfile
	Impl     any
}

type Registry struct {
	entries   []RoleEntry
	providers map[ProviderID]IdentityProvider
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) RegisterProvider(provider IdentityProvider) error {
	errb := oops.In("provider_registry")
	if provider == nil {
		return errb.New("register provider: provider identity is nil")
	}

	value := reflect.ValueOf(provider)
	if isNilReflectValue(value) {
		return errb.New("register provider: provider identity is nil")
	}
	identity := provider.ProviderIdentity()
	if identity.ID == "" {
		return errb.New("register provider: provider id is empty")
	}
	if r.providers == nil {
		r.providers = map[ProviderID]IdentityProvider{}
	}
	r.providers[identity.ID] = provider

	value = reflect.Indirect(value)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return errb.With("provider", identity.ID).New("register provider: provider must be a struct or struct pointer")
	}

	if registrationProvider, ok := provider.(RoleRegistrationProvider); ok {
		registrations := registrationProvider.RoleRegistrations()
		if len(registrations) == 0 {
			return errb.With("provider", identity.ID).New("register provider: provider has no role registrations")
		}
		return r.Register(provider, registrations...)
	}

	registrations, err := collectEmbeddedRoleRegistrations(value, identity)
	if err != nil {
		return err
	}
	if len(registrations) == 0 {
		return errb.With("provider", identity.ID).New("register provider: provider has no role fields")
	}

	return r.Register(provider, registrations...)
}

func (r *Registry) Provider(id ProviderID) (IdentityProvider, bool) {
	if r == nil || r.providers == nil {
		return nil, false
	}
	provider, ok := r.providers[id]
	return provider, ok
}

func collectEmbeddedRoleRegistrations(value reflect.Value, identity Identity) ([]RoleRegistration, error) {
	errb := oops.In("provider_registry")
	roleProviderType := reflect.TypeOf((*RoleProvider)(nil)).Elem()
	registrations := make([]RoleRegistration, 0)
	for i := 0; i < value.NumField(); i++ {
		fieldInfo := value.Type().Field(i)
		if !fieldInfo.Anonymous {
			continue
		}
		if !fieldInfo.IsExported() {
			continue
		}
		field := value.Field(i)
		if !field.CanInterface() {
			continue
		}
		if !field.Type().Implements(roleProviderType) {
			continue
		}
		if isNilReflectValue(field) {
			return nil, errb.With("provider", identity.ID, "field", fieldInfo.Name).New("register provider: embedded role field is nil")
		}
		roleProvider, ok := field.Interface().(RoleProvider)
		if !ok {
			return nil, errb.With("provider", identity.ID, "field", fieldInfo.Name).New("register provider: embedded role field does not implement RoleProvider")
		}
		registrations = append(registrations, roleProvider.RoleRegistration())
	}
	return registrations, nil
}

func (r *Registry) Register(provider IdentityProvider, roles ...RoleRegistration) error {
	errb := oops.In("provider_registry")
	if provider == nil {
		return errb.New("register provider role: provider identity is nil")
	}
	identity := provider.ProviderIdentity()
	if identity.ID == "" {
		return errb.New("register provider role: provider id is empty")
	}
	providerErrb := errb.With("provider", identity.ID)
	for _, role := range roles {
		if role.Profile.Role == "" {
			return providerErrb.New("register provider role: role is empty")
		}
		if role.Profile.Compatibility.DataLatency == "" {
			return providerErrb.With("role", role.Profile.Role).New("register provider role: data compatibility is required")
		}
		if role.Impl == nil {
			return providerErrb.With("role", role.Profile.Role).New("register provider role: implementation is nil")
		}
		r.entries = append(r.entries, RoleEntry{
			Provider: identity,
			Profile:  role.Profile,
			Impl:     role.Impl,
		})
	}
	return nil
}

func (r *Registry) RegisterConfigured(opts RegisterOptions, config Config, builders ...ProviderBuilder) error {
	errb := oops.In("provider_registry")
	if r == nil {
		return errb.New("register configured providers: registry is nil")
	}
	for _, builder := range builders {
		if builder == nil {
			return errb.New("register configured providers: builder is nil")
		}
		if err := r.registerConfiguredProvider(opts, config, builder); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) RegisterConfiguredFromEnv(opts RegisterOptions, builders ...ProviderBuilder) error {
	return r.RegisterConfigured(opts, ConfigFromEnv(), builders...)
}

func (r *Registry) registerConfiguredProvider(opts RegisterOptions, config Config, builder ProviderBuilder) error {
	providerID := builder.ID()
	errb := oops.In("provider_registry").With("provider", providerID)
	if providerID == "" {
		return errb.New("register configured provider: builder id is empty")
	}

	decision := builder.Decide(opts, config)
	if !decision.Register {
		return nil
	}

	providerInstance, err := builder.Build(config)
	errb = errb.With("reason", decision.Reason)
	if err != nil {
		return errb.Wrapf(err, "build configured provider reason=%s", decision.Reason)
	}
	if providerInstance == nil {
		return errb.New("build configured provider returned nil")
	}

	identity := providerInstance.ProviderIdentity()
	if identity.ID != providerID {
		return errb.With("actual_provider", identity.ID).New("configured provider builder returned mismatched provider id")
	}
	if err := r.RegisterProvider(providerInstance); err != nil {
		return errb.Wrapf(err, "register configured provider")
	}
	return nil
}

func (r *Registry) Entries(role Role) []RoleEntry {
	entries := make([]RoleEntry, 0)
	for _, entry := range r.entries {
		if entry.Profile.Role == role {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (r *Registry) Roles() []Role {
	seen := make(map[Role]struct{})
	roles := make([]Role, 0)
	for _, entry := range r.entries {
		role := entry.Profile.Role
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i] < roles[j]
	})
	return roles
}

func isNilReflectValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
