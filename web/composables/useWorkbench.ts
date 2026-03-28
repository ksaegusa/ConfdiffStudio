export interface PairInput {
  name: string;
  before: string;
  after: string;
  beforeFileName?: string;
  afterFileName?: string;
}

export function useWorkbench() {
  const pairs = useState<PairInput[]>("wb:pairs", () => [
    {
      name: "",
      before: "",
      after: "",
      beforeFileName: "",
      afterFileName: "",
    },
  ]);

  function addPair() {
    pairs.value.push({
      name: "",
      before: "",
      after: "",
      beforeFileName: "",
      afterFileName: "",
    });
  }

  function removePair(index: number) {
    pairs.value.splice(index, 1);
  }

  function replacePair(index: number, pair: PairInput) {
    pairs.value[index] = pair;
  }

  return {
    pairs,
    addPair,
    removePair,
    replacePair,
  };
}
