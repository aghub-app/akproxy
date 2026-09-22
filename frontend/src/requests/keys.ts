export const queryKeys = {
  home: {
    all: () => ["home"] as const,
    snapshot: () => ["home", "snapshot"] as const,
    models: (address: string | null) => ["home", "models", address] as const,
  },
};
