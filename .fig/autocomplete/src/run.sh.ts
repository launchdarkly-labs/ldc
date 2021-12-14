const configArgs: Fig.Arg[] = [
  {
    name: "config name",
  },
  {
    name: "api token",
  },
  {
    name: "project",
    description: "default project key",
  },
  {
    name: "environment",
    description: "default environment key",
  },
  {
    name: "server url",
    isOptional: true,
  },
];

const completionSpec: Fig.Spec = {
  name: "ldc",
  description: "ldc is a command-line api client for LaunchDarkly",
  subcommands: [
    {
      name: "configs",
      description: "Manage configurations",
      subcommands: [
        {
          name: "add",
          description: "add config <config name> <api token> <project> <environment> [server url]",
          args: configArgs,
        },
        {
          name: "set",
          description: "Change configuration <config name>",
          args: [
            {
              name: "config name",
            },
          ],
        },
        {
          name: "rename",
          description: "Rename config <config name> <new name>",
          args: [
            {
              name: "config name",
            },
            {
              name: "new name",
            },
          ],
        },
        {
          name: "edit",
          description: "Update config: <config name> <api token> <project> <environment> [server url]",
          args: configArgs,
        },
        {
          name: "rm",
          description: "Remove config: <config name>",
          args: [
            {
              name: "config name",
            },
          ],
        },
      ],
    }
  ],
};
export default completionSpec;