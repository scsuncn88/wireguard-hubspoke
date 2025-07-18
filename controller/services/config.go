package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wg-hubspoke/wg-hubspoke/controller/models"
	"gorm.io/gorm"
	"gopkg.in/yaml.v2"
)

type ConfigService struct {
	db          *gorm.DB
	auditService *AuditService
}

type ConfigExport struct {
	Version     string                 `json:"version" yaml:"version"`
	ExportedAt  time.Time              `json:"exported_at" yaml:"exported_at"`
	ExportedBy  uuid.UUID              `json:"exported_by" yaml:"exported_by"`
	Nodes       []models.Node          `json:"nodes" yaml:"nodes"`
	Policies    []models.Policy        `json:"policies" yaml:"policies"`
	Users       []UserExport           `json:"users" yaml:"users"`
	Topology    *models.Topology       `json:"topology" yaml:"topology"`
	SystemConfig map[string]interface{} `json:"system_config" yaml:"system_config"`
}

type UserExport struct {
	ID        uuid.UUID `json:"id" yaml:"id"`
	Username  string    `json:"username" yaml:"username"`
	Email     string    `json:"email" yaml:"email"`
	Role      string    `json:"role" yaml:"role"`
	Active    bool      `json:"active" yaml:"active"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
	// Note: Password hashes are NOT exported for security reasons
}

type ImportOptions struct {
	OverwriteExisting bool   `json:"overwrite_existing" yaml:"overwrite_existing"`
	SkipUsers         bool   `json:"skip_users" yaml:"skip_users"`
	SkipNodes         bool   `json:"skip_nodes" yaml:"skip_nodes"`
	SkipPolicies      bool   `json:"skip_policies" yaml:"skip_policies"`
	ImportedBy        uuid.UUID `json:"imported_by" yaml:"imported_by"`
}

type ImportResult struct {
	Success        bool                   `json:"success"`
	NodesImported  int                    `json:"nodes_imported"`
	NodesSkipped   int                    `json:"nodes_skipped"`
	NodesErrors    []string               `json:"nodes_errors"`
	UsersImported  int                    `json:"users_imported"`
	UsersSkipped   int                    `json:"users_skipped"`
	UsersErrors    []string               `json:"users_errors"`
	PoliciesImported int                  `json:"policies_imported"`
	PoliciesSkipped  int                  `json:"policies_skipped"`
	PoliciesErrors   []string             `json:"policies_errors"`
	GeneralErrors    []string             `json:"general_errors"`
	ImportedAt       time.Time            `json:"imported_at"`
}

func NewConfigService(db *gorm.DB, auditService *AuditService) *ConfigService {
	return &ConfigService{
		db:          db,
		auditService: auditService,
	}
}

func (s *ConfigService) ExportConfiguration(ctx context.Context, exportedBy uuid.UUID, format string) ([]byte, error) {
	// Create export structure
	export := &ConfigExport{
		Version:    "1.0",
		ExportedAt: time.Now(),
		ExportedBy: exportedBy,
		SystemConfig: map[string]interface{}{
			"export_format": format,
			"database_version": "1.0",
		},
	}

	// Export nodes
	var nodes []models.Node
	if err := s.db.Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("failed to export nodes: %w", err)
	}
	export.Nodes = nodes

	// Export policies
	var policies []models.Policy
	if err := s.db.Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("failed to export policies: %w", err)
	}
	export.Policies = policies

	// Export users (without passwords)
	var users []models.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to export users: %w", err)
	}
	
	export.Users = make([]UserExport, len(users))
	for i, user := range users {
		export.Users[i] = UserExport{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      string(user.Role),
			Active:    user.Active,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	// Export topology
	var topology models.Topology
	if err := s.db.First(&topology).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to export topology: %w", err)
	}
	if err != gorm.ErrRecordNotFound {
		export.Topology = &topology
	}

	// Log export action
	s.auditService.LogAction(ctx, exportedBy, "export_configuration", "configuration", uuid.Nil, 
		map[string]interface{}{
			"format": format,
			"nodes_count": len(nodes),
			"policies_count": len(policies),
			"users_count": len(users),
		})

	// Serialize based on format
	switch format {
	case "json":
		return json.MarshalIndent(export, "", "  ")
	case "yaml":
		return yaml.Marshal(export)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

func (s *ConfigService) ImportConfiguration(ctx context.Context, data []byte, format string, options ImportOptions) (*ImportResult, error) {
	result := &ImportResult{
		ImportedAt: time.Now(),
		NodesErrors: []string{},
		UsersErrors: []string{},
		PoliciesErrors: []string{},
		GeneralErrors: []string{},
	}

	// Parse configuration data
	var config ConfigExport
	var err error
	
	switch format {
	case "json":
		err = json.Unmarshal(data, &config)
	case "yaml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported import format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Start database transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Import nodes
	if !options.SkipNodes {
		nodesImported, nodesSkipped, nodeErrors := s.importNodes(tx, config.Nodes, options.OverwriteExisting)
		result.NodesImported = nodesImported
		result.NodesSkipped = nodesSkipped
		result.NodesErrors = nodeErrors
	}

	// Import policies
	if !options.SkipPolicies {
		policiesImported, policiesSkipped, policyErrors := s.importPolicies(tx, config.Policies, options.OverwriteExisting)
		result.PoliciesImported = policiesImported
		result.PoliciesSkipped = policiesSkipped
		result.PoliciesErrors = policyErrors
	}

	// Import users
	if !options.SkipUsers {
		usersImported, usersSkipped, userErrors := s.importUsers(tx, config.Users, options.OverwriteExisting)
		result.UsersImported = usersImported
		result.UsersSkipped = usersSkipped
		result.UsersErrors = userErrors
	}

	// Import topology
	if config.Topology != nil {
		if err := s.importTopology(tx, config.Topology, options.OverwriteExisting); err != nil {
			result.GeneralErrors = append(result.GeneralErrors, fmt.Sprintf("Topology import failed: %v", err))
		}
	}

	// Check if there were any critical errors
	if len(result.GeneralErrors) > 0 || (len(result.NodesErrors) > 0 && !options.SkipNodes) {
		tx.Rollback()
		result.Success = false
		return result, fmt.Errorf("import failed due to critical errors")
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		result.Success = false
		result.GeneralErrors = append(result.GeneralErrors, fmt.Sprintf("Transaction commit failed: %v", err))
		return result, fmt.Errorf("failed to commit import transaction: %w", err)
	}

	result.Success = true

	// Log import action
	s.auditService.LogAction(ctx, options.ImportedBy, "import_configuration", "configuration", uuid.Nil,
		map[string]interface{}{
			"format": format,
			"nodes_imported": result.NodesImported,
			"policies_imported": result.PoliciesImported,
			"users_imported": result.UsersImported,
			"overwrite_existing": options.OverwriteExisting,
		})

	return result, nil
}

func (s *ConfigService) importNodes(tx *gorm.DB, nodes []models.Node, overwrite bool) (int, int, []string) {
	imported := 0
	skipped := 0
	errors := []string{}

	for _, node := range nodes {
		// Check if node exists
		var existingNode models.Node
		err := tx.Where("id = ? OR name = ?", node.ID, node.Name).First(&existingNode).Error
		
		if err == nil {
			// Node exists
			if overwrite {
				// Update existing node
				if err := tx.Model(&existingNode).Updates(node).Error; err != nil {
					errors = append(errors, fmt.Sprintf("Failed to update node %s: %v", node.Name, err))
					continue
				}
				imported++
			} else {
				skipped++
			}
		} else if err == gorm.ErrRecordNotFound {
			// Node doesn't exist, create new
			if err := tx.Create(&node).Error; err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create node %s: %v", node.Name, err))
				continue
			}
			imported++
		} else {
			errors = append(errors, fmt.Sprintf("Database error for node %s: %v", node.Name, err))
		}
	}

	return imported, skipped, errors
}

func (s *ConfigService) importPolicies(tx *gorm.DB, policies []models.Policy, overwrite bool) (int, int, []string) {
	imported := 0
	skipped := 0
	errors := []string{}

	for _, policy := range policies {
		// Check if policy exists
		var existingPolicy models.Policy
		err := tx.Where("id = ? OR name = ?", policy.ID, policy.Name).First(&existingPolicy).Error
		
		if err == nil {
			// Policy exists
			if overwrite {
				// Update existing policy
				if err := tx.Model(&existingPolicy).Updates(policy).Error; err != nil {
					errors = append(errors, fmt.Sprintf("Failed to update policy %s: %v", policy.Name, err))
					continue
				}
				imported++
			} else {
				skipped++
			}
		} else if err == gorm.ErrRecordNotFound {
			// Policy doesn't exist, create new
			if err := tx.Create(&policy).Error; err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create policy %s: %v", policy.Name, err))
				continue
			}
			imported++
		} else {
			errors = append(errors, fmt.Sprintf("Database error for policy %s: %v", policy.Name, err))
		}
	}

	return imported, skipped, errors
}

func (s *ConfigService) importUsers(tx *gorm.DB, users []UserExport, overwrite bool) (int, int, []string) {
	imported := 0
	skipped := 0
	errors := []string{}

	for _, userExport := range users {
		// Check if user exists
		var existingUser models.User
		err := tx.Where("id = ? OR username = ? OR email = ?", userExport.ID, userExport.Username, userExport.Email).First(&existingUser).Error
		
		if err == nil {
			// User exists
			if overwrite {
				// Update existing user (except password)
				updates := map[string]interface{}{
					"username":   userExport.Username,
					"email":      userExport.Email,
					"role":       userExport.Role,
					"active":     userExport.Active,
					"updated_at": time.Now(),
				}
				if err := tx.Model(&existingUser).Updates(updates).Error; err != nil {
					errors = append(errors, fmt.Sprintf("Failed to update user %s: %v", userExport.Username, err))
					continue
				}
				imported++
			} else {
				skipped++
			}
		} else if err == gorm.ErrRecordNotFound {
			// User doesn't exist, create new (with default password)
			newUser := models.User{
				ID:       userExport.ID,
				Username: userExport.Username,
				Email:    userExport.Email,
				Role:     models.UserRole(userExport.Role),
				Active:   userExport.Active,
				Password: "$2a$10$defaulthashedpassword", // Default password, user must change
			}
			if err := tx.Create(&newUser).Error; err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create user %s: %v", userExport.Username, err))
				continue
			}
			imported++
		} else {
			errors = append(errors, fmt.Sprintf("Database error for user %s: %v", userExport.Username, err))
		}
	}

	return imported, skipped, errors
}

func (s *ConfigService) importTopology(tx *gorm.DB, topology *models.Topology, overwrite bool) error {
	var existingTopology models.Topology
	err := tx.First(&existingTopology).Error
	
	if err == nil {
		// Topology exists
		if overwrite {
			return tx.Model(&existingTopology).Updates(topology).Error
		}
		return nil // Skip if not overwriting
	} else if err == gorm.ErrRecordNotFound {
		// Topology doesn't exist, create new
		return tx.Create(topology).Error
	}
	
	return err
}

func (s *ConfigService) ValidateConfiguration(ctx context.Context, data []byte, format string) ([]string, error) {
	warnings := []string{}
	
	// Parse configuration data
	var config ConfigExport
	var err error
	
	switch format {
	case "json":
		err = json.Unmarshal(data, &config)
	case "yaml":
		err = yaml.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate version compatibility
	if config.Version != "1.0" {
		warnings = append(warnings, fmt.Sprintf("Configuration version %s may not be fully compatible with current system version", config.Version))
	}

	// Validate nodes
	nodeNames := make(map[string]bool)
	for _, node := range config.Nodes {
		if nodeNames[node.Name] {
			warnings = append(warnings, fmt.Sprintf("Duplicate node name: %s", node.Name))
		}
		nodeNames[node.Name] = true
		
		if node.PublicKey == "" {
			warnings = append(warnings, fmt.Sprintf("Node %s has empty public key", node.Name))
		}
		
		if node.AllocatedIP == "" {
			warnings = append(warnings, fmt.Sprintf("Node %s has empty allocated IP", node.Name))
		}
	}

	// Validate users
	usernames := make(map[string]bool)
	emails := make(map[string]bool)
	for _, user := range config.Users {
		if usernames[user.Username] {
			warnings = append(warnings, fmt.Sprintf("Duplicate username: %s", user.Username))
		}
		usernames[user.Username] = true
		
		if emails[user.Email] {
			warnings = append(warnings, fmt.Sprintf("Duplicate email: %s", user.Email))
		}
		emails[user.Email] = true
	}

	// Validate policies
	policyNames := make(map[string]bool)
	for _, policy := range config.Policies {
		if policyNames[policy.Name] {
			warnings = append(warnings, fmt.Sprintf("Duplicate policy name: %s", policy.Name))
		}
		policyNames[policy.Name] = true
	}

	return warnings, nil
}

func (s *ConfigService) GetConfigurationSummary(ctx context.Context) (map[string]interface{}, error) {
	summary := make(map[string]interface{})
	
	// Count nodes
	var nodeCount int64
	if err := s.db.Model(&models.Node{}).Count(&nodeCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count nodes: %w", err)
	}
	summary["nodes_count"] = nodeCount
	
	// Count policies
	var policyCount int64
	if err := s.db.Model(&models.Policy{}).Count(&policyCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count policies: %w", err)
	}
	summary["policies_count"] = policyCount
	
	// Count users
	var userCount int64
	if err := s.db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}
	summary["users_count"] = userCount
	
	// Get system info
	summary["system_version"] = "1.0"
	summary["database_version"] = "1.0"
	summary["last_updated"] = time.Now()
	
	return summary, nil
}