import { createSecurityApi, createSecurityMutateApi } from './shared';

export const DescribeAIAgentSkillList = createSecurityApi('DescribeAIAgentSkillList', 'csip');

export const DescribeAIAgentAutoOpenConfig = createSecurityApi('DescribeAIAgentAutoOpenConfig', 'cwp');

export const DescribeLicenseBindList = createSecurityApi('DescribeLicenseBindList', 'cwp');

export const DescribeLicenseList = createSecurityApi('DescribeLicenseList', 'cwp');

export const ModifyAIAgentAutoOpenConfig = createSecurityMutateApi('ModifyAIAgentAutoOpenConfig', 'cwp');
