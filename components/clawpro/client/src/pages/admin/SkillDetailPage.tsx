import { useLocation } from 'wouter';
import SkillDetail from './SkillLibrary/SkillDetail';

interface SkillDetailPageProps {
  skillId: string;
}

export default function SkillDetailPage({ skillId }: SkillDetailPageProps) {
  const [, setLocation] = useLocation();

  const handleBack = () => {
    setLocation('/admin/skill-config');
  };

  return <SkillDetail skillId={skillId} onBack={handleBack} />;
}
