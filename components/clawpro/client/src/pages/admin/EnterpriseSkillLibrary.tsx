import SkillListTab from './SkillLibrary/SkillListTab';

interface EnterpriseSkillLibraryProps {
  securityServiceActive?: boolean;
}

export default function EnterpriseSkillLibrary({ securityServiceActive }: EnterpriseSkillLibraryProps) {
  return (
    <div className="page-enter">
      <SkillListTab securityServiceActive={securityServiceActive} />
    </div>
  );
}
