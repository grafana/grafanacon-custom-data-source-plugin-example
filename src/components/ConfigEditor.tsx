import React, { ChangeEvent } from 'react';
import { InlineField, Input, SecretInput } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MyDataSourceOptions, MySecureJsonData } from '../types';

interface Props extends DataSourcePluginOptionsEditorProps<MyDataSourceOptions, MySecureJsonData> {}

export function ConfigEditor(props: Props) {
  const { onOptionsChange, options } = props;
  const { jsonData, secureJsonFields, secureJsonData } = options;

  const onDomainChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        domain: event.target.value,
      },
    });
  };

  const onResourceChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        resource: event.target.value,
      },
    });
  };

  // Secure field (only sent to the backend)
  const onAppTokenChange = (event: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({
      ...options,
      secureJsonData: {
        appToken: event.target.value,
      },
    });
  };

  const onResetAppToken = () => {
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        appToken: false,
      },
      secureJsonData: {
        ...options.secureJsonData,
        appToken: '',
      },
    });
  };

  return (
    <>
      <InlineField label="Domain" labelWidth={14} interactive tooltip={'Json field returned to frontend'}>
        <Input
          id="config-editor-domain"
          onChange={onDomainChange}
          value={jsonData.domain}
          placeholder="Enter the domain, e.g. http://host.docker.internal:8080 or https://data.seattle.gov"
          width={40}
        />
      </InlineField>
      <InlineField label="Resource" labelWidth={14} interactive tooltip={'Json field returned to frontend'}>
        <Input
          id="config-editor-resource"
          onChange={onResourceChange}
          value={jsonData.resource}
          placeholder="Enter the resource, e.g. egc4-d24i"
          width={40}
        />
      </InlineField>
      <InlineField label="App Token" labelWidth={14} interactive tooltip={'Secure json field (backend only)'}>
        <SecretInput
          required
          id="config-editor-app-token"
          isConfigured={secureJsonFields.appToken}
          value={secureJsonData?.appToken}
          placeholder="Enter your App Token"
          width={40}
          onReset={onResetAppToken}
          onChange={onAppTokenChange}
        />
      </InlineField>
    </>
  );
}
