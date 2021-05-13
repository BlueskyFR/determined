import { Button } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import InfoBox from 'components/InfoBox';
import Section from 'components/Section';
import {
  CheckpointDetail, CheckpointState, CheckpointWorkload, ExperimentBase, TrialDetails,
} from 'types';
import { formatDatetime } from 'utils/date';
import { capitalize, humanReadableBytes } from 'utils/string';
import { shortEnglishHumannizer } from 'utils/time';
import { trialDurations } from 'utils/trial';
import { checkpointSize } from 'utils/types';

import css from './TrialInfoBox.module.scss';

interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

const TrialInfoBox: React.FC<Props> = ({ trial, experiment }: Props) => {
  const [ showBestCheckpoint, setShowBestCheckpoint ] = useState(false);

  const { metric } = experiment.config.searcher || {};

  const bestValidation = useMemo(() => {
    return (trial.bestValidationMetric?.metrics || {})[metric];
  }, [ metric, trial.bestValidationMetric?.metrics ]);

  const bestCheckpoint: CheckpointDetail | undefined = useMemo(() => {
    const cp = trial.bestAvailableCheckpoint;
    if (!cp) return;

    return {
      ...cp,
      batch: cp.totalBatches,
      experimentId: trial.experimentId,
      trialId: trial.id,
    };
  }, [ trial.bestAvailableCheckpoint, trial.experimentId, trial.id ]);

  const totalCheckpointsSize = useMemo(() => {
    const totalBytes = trial.workloads
      .filter(step => step.checkpoint
        && step.checkpoint.state === CheckpointState.Completed)
      .map(step =>checkpointSize(step.checkpoint as CheckpointWorkload))
      .reduce((acc, cur) => acc + cur, 0);
    return humanReadableBytes(totalBytes);
  }, [ trial.workloads ]);

  const durations = useMemo(() => trialDurations(trial.workloads), [ trial.workloads ]);

  const handleShowBestCheckpoint = useCallback(() => setShowBestCheckpoint(true), []);
  const handleHideBestCheckpoint = useCallback(() => setShowBestCheckpoint(false), []);

  const workloadStatus: string | undefined =
    Object.entries(trial.workloads.last()).find(e => !!e[1])?.first();

  const infoRows = [
    {
      content: formatDatetime(trial.startTime),
      label: 'Start Time',
    },
    {
      content: trial.endTime && formatDatetime(trial.endTime),
      label: 'End Time',
    },
    {
      content: <div className={css.duration}>
        <div>Training: {shortEnglishHumannizer(durations.train)}</div>
        <div>Checkpointing: {shortEnglishHumannizer(durations.checkpoint)}</div>
        <div>Validating: {shortEnglishHumannizer(durations.validation)}</div>
      </div>,
      label: 'Durations',
    },
    {
      content: (trial.state === 'ACTIVE' && workloadStatus !== undefined) &&
      `${capitalize(workloadStatus)} on batch ${trial.totalBatchesProcessed}`,
      label: 'Current Workload',
    },
    {
      content: bestValidation &&
        <>
          <HumanReadableFloat num={bestValidation} /> {`(${experiment.config.searcher.metric})`}
        </>,
      label: 'Best Validation',
    },
    {
      content: bestCheckpoint && (<>
        <Button onClick={handleShowBestCheckpoint}>
          Trial {bestCheckpoint.trialId} Batch {bestCheckpoint.batch}
        </Button>
        <CheckpointModal
          checkpoint={bestCheckpoint}
          config={experiment.config}
          show={showBestCheckpoint}
          title={`Best Checkpoint for Trial ${trial.id}`}
          onHide={handleHideBestCheckpoint} />
      </>),
      label: 'Best Checkpoint',
    },
    {
      content: totalCheckpointsSize,
      label: 'Total Checkpoint Size',
    },
  ];

  return (
    <Section bodyBorder maxHeight title="Summary">
      <InfoBox rows={infoRows} />
    </Section>
  );
};

export default TrialInfoBox;
