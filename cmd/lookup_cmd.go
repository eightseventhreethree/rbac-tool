package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/rbac"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func NewCommandLookup() *cobra.Command {

	clusterContext := ""
	regex := ""
	// Support overrides
	cmd := &cobra.Command{
		Use:     "lookup",
		Aliases: []string{"look"},
		Short:   "RBAC Lookup by subject (user/group/serviceaccount) name",
		Long: `
A Kubernetes RBAC lookup of Roles/ClusterRoles used by a given User/ServiceAccount/Group

Examples:

# Search All Service Accounts
rbac-tool lookup -e '.*'

# Search All Service Accounts That Contains myname
rbac-tool lookup -e '.*myname.*'

# Lookup System Accounts (all accounts that starts with system: )
rbac-tool lookup -e '^system:.*'

`,
		Hidden: false,
		RunE: func(c *cobra.Command, args []string) error {
			var re *regexp.Regexp
			var err error

			if regex != "" {
				re, err = regexp.Compile(regex)
			} else {
				if len(args) != 1 {
					re, err = regexp.Compile(fmt.Sprintf(`.*`))
				} else {
					re, err = regexp.Compile(fmt.Sprintf(`(?mi)%v`, args[0]))
				}
			}

			if err != nil {
				return err
			}

			client, err := kube.NewClient(clusterContext)
			if err != nil {
				return fmt.Errorf("Failed to create kubernetes client - %v", err)
			}

			perms, err := rbac.NewPermissionsFromCluster(client)
			if err != nil {
				return err
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"SUBJECT", "SUBJECT TYPE", "SCOPE", "NAMESPACE", "ROLE"})
			table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
			table.SetBorder(false)
			table.SetAlignment(tablewriter.ALIGN_LEFT)

			rows := [][]string{}
			for _, bindings := range perms.RoleBindings {
				for _, binding := range bindings {
					for _, subject := range binding.Subjects {
						if !re.MatchString(subject.Name) {
							continue
						}

						//Subject match
						_, exist := perms.Roles[binding.Namespace]
						if !exist {
							continue
						}

						if binding.Namespace == "" {
							row := []string{subject.Name, subject.Kind, "ClusterRole", "", binding.RoleRef.Name}
							rows = append(rows, row)
						} else {
							row := []string{subject.Name, subject.Kind, "Role", binding.Namespace, binding.RoleRef.Name}
							rows = append(rows, row)
						}

					}
				}
			}

			sort.Slice(rows, func(i, j int) bool {
				if strings.Compare(rows[i][0], rows[j][0]) == 0 {
					return (strings.Compare(rows[i][3], rows[j][3]) < 0)
				}

				return (strings.Compare(rows[i][0], rows[j][0]) < 0)
			})

			table.AppendBulk(rows)
			table.Render()

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&clusterContext, "cluster-context", "", "Cluster Context .use 'kubectl config get-contexts' to list available contexts")
	flags.StringVarP(&regex, "regex", "e", "", "Specify whether run the lookup using a regex match")

	return cmd
}