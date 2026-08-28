import React from 'react';

/**
 * PersonaMem 评测结果表格
 * 
 * 展示 Agent 原生 vs TDAI-Memory 的性能对比
 */
export const BenchmarkTable: React.FC = () => {
  const benchmarkData = [
    {
      metric: '召回用户事实',
      native: '29.63%',
      memory: '79.07%',
    },
    {
      metric: '个性化推荐',
      native: '46.67%',
      memory: '76.36%',
    },
    {
      metric: '偏好演变跟踪',
      native: '66.67%',
      memory: '83.45%',
    },
    {
      metric: '场景泛化',
      native: '31.58%',
      memory: '78.95%',
    },
    {
      metric: '总准确率',
      native: '47.85%',
      memory: '76.10%',
    },
  ];

  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse rounded-[12px] overflow-hidden border border-[#E8EAF0]">
        <thead>
          <tr>
            <th className="px-[18px] py-[14px] text-left text-[13.5px] font-semibold bg-[#F9FAFB] text-[#6B7280] w-[34%]">
              评测项
            </th>
            <th className="px-[18px] py-[14px] text-center text-[13.5px] font-semibold bg-[#F0FDF4] text-[#16A34A]">
              Agent 原生
            </th>
            <th className="px-[18px] py-[14px] text-center text-[13.5px] font-semibold bg-[#FAF5FF] text-[#7C3AED]">
              TDAI-Memory
            </th>
          </tr>
        </thead>
        <tbody>
          {benchmarkData.map((row, index) => (
            <tr key={index} className="border-t border-[#F0F0F5]">
              <td className="px-[18px] py-[12px] text-left text-[13px] font-medium text-[#374151] bg-[#FAFBFE]">
                {row.metric}
              </td>
              <td className="px-[18px] py-[12px] text-center text-[13px] text-[#5c5c7a] bg-white">
                {row.native}
              </td>
              <td className="px-[18px] py-[12px] text-center text-[13px] font-semibold text-[#7C3AED] bg-white">
                {row.memory}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
