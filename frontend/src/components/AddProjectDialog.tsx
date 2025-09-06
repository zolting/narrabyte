import React from "react";
import { Input } from "@/components/ui/input";
import { useTranslation } from "react-i18next";
import DirectoryPicker from "@/components/DirectoryPicker";
import { Button } from "@/components/ui/button";

type AddProjectDialogProps = {
  open: boolean;
  onClose: () => void;
  onSubmit: (data: { name: string; docDirectory: string; codebaseDirectory: string }) => void;
};

export const AddProjectDialog: React.FC<AddProjectDialogProps> = ({
  open,
  onClose,
  onSubmit,
}) => {
  const { t } = useTranslation();
  const [name, setName] = React.useState("");
  const [docDirectory, setDocDirectory] = React.useState("");
  const [codebaseDirectory, setCodebaseDirectory] = React.useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({ name, docDirectory, codebaseDirectory });
  };

  React.useEffect(() => {
    if (open) {
      setName("");
      setDocDirectory("");
      setCodebaseDirectory("");
    }
  }, [open]);

  if (!open) return null;

  return (
    <div className="dialog-backdrop">
      <div className="dialog">
        <h2 className="mb-4 text-lg font-bold">{t("projectManager.addProject")}</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block mb-1 font-medium">{t("projectManager.projectName")}</label>
            <Input
              value={name}
              onChange={e => setName(e.target.value)}
              required
              placeholder="Nom du projet"
            />
          </div>
          <div>
            <label className="block mb-1 font-medium">{t("projectManager.docDirectory")}</label>
            <DirectoryPicker onDirectorySelected={setDocDirectory} />
            {docDirectory && <div className="text-xs mt-1">{docDirectory}</div>}
          </div>
          <div>
            <label className="block mb-1 font-medium">{t("projectManager.codebaseDirectory")}</label>
            <DirectoryPicker onDirectorySelected={setCodebaseDirectory} />
            {codebaseDirectory && <div className="text-xs mt-1">{codebaseDirectory}</div>}
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onClose}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={!(name && docDirectory && codebaseDirectory)}>
              {t("home.addProject")}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};