import { useState } from 'react';
import { Layout, Skeleton } from 'antd';
import Button from 'antd-button-color';
import { StoryContext } from '@storybook/react/dist/ts3.9/client/preview/types';
import { Story } from '@storybook/react';

const ModalDecorator = (Story: Story, context: StoryContext) => {
  const [show, setShow] = useState(false);
  context.args.showModal = show;
  context.args.setShowModal = setShow;

  return (
    <Layout>
      <Layout.Content>
        <Story />
        <div className="m-4">
          <Skeleton />
        </div>
        <div className=" flex justify-center mt-12">
          <Button
            onClick={() => setShow(true)}
            type="primary"
            shape="round"
            size="large"
          >
            Open Modal
          </Button>
        </div>
      </Layout.Content>
    </Layout>
  );
};

export { ModalDecorator };
