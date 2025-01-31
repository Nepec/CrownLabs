import TemplatesTable, { ITemplatesTableProps } from './TemplatesTable';
import { Story, Meta } from '@storybook/react';
import { someKeysOf } from '../../../../utils';

export default {
  title: 'Components/workspaces/Templates/TemplatesTable',
  component: TemplatesTable,
  argTypes: {},
} as Meta;

const defaultArgs: someKeysOf<ITemplatesTableProps> = {
  templates: [
    {
      id: '0_1',
      name: 'Ubuntu VM',
      gui: true,
      instances: [
        { id: 1, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
        { id: 2, name: 'Ubuntu VM', ip: '192.168.0.1', status: false },
      ],
    },
    {
      id: '0_2',
      name: 'Ubuntu VM',
      gui: false,
      instances: [
        { id: 1, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
        { id: 2, name: 'Ubuntu VM', ip: '192.168.0.1', status: false },
      ],
    },
    {
      id: '0_3',
      name: 'Windows VM',
      gui: true,
      instances: [
        { id: 1, name: 'Windows VM', ip: '192.168.0.1', status: true },
      ],
    },
    {
      id: '0_4',
      name: 'Console (Linux)',
      gui: false,
      instances: [
        { id: 1, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
        { id: 2, name: 'Ubuntu VM', ip: '192.168.0.1', status: false },
      ],
    },
    {
      id: '0_5',
      name: 'Ubuntu VM',
      gui: true,
      instances: [
        { id: 1, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
        { id: 2, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
        { id: 3, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
      ],
    },
    {
      id: '0_6',
      name: 'Ubuntu VM',
      gui: true,
      instances: [
        { id: 1, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
        { id: 2, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
        { id: 3, name: 'Ubuntu VM', ip: '192.168.0.1', status: true },
      ],
    },
  ],
  role: 'manager',
  editTemplate: () => null,
  deleteTemplate: () => null,
};

const Template: Story<ITemplatesTableProps> = args => (
  <TemplatesTable {...args} />
);

export const Expandable = Template.bind({});

Expandable.args = defaultArgs;

export const NotExpandable = Template.bind({});

NotExpandable.args = {
  ...defaultArgs,
  templates: [
    {
      id: '0_1',
      name: 'Ubuntu VM',
      gui: true,
      instances: [],
    },
    {
      id: '0_2',
      name: 'Ubuntu VM',
      gui: false,
      instances: [],
    },
    {
      id: '0_3',
      name: 'Windows VM',
      gui: true,
      instances: [],
    },
    {
      id: '0_4',
      name: 'Console (Linux)',
      gui: false,
      instances: [],
    },
    {
      id: '0_5',
      name: 'Ubuntu VM',
      gui: true,
      instances: [],
    },
    {
      id: '0_6',
      name: 'Ubuntu VM',
      gui: true,
      instances: [],
    },
  ],
};
