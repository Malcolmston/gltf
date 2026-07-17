import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';
import type { DocIndex } from 'go-ui';

// A minimal DocIndex the stubbed fetch returns for DocsApp's doc.json request.
const DOC_INDEX: DocIndex = {
  module: 'github.com/malcolmston/gltf',
  packages: [
    {
      importPath: 'github.com/malcolmston/gltf',
      name: 'gltf',
      synopsis: 'Package gltf is a dependency-free toolkit for reading, writing, and inspecting glTF 2.0 assets.',
      doc: 'Package gltf is a dependency-free toolkit for reading, writing, and inspecting glTF 2.0 assets.',
      consts: [],
      vars: [],
      types: [
        {
          name: 'Document',
          signature: 'type Document struct{}',
          doc: 'Document is the root object and mirrors the glTF 2.0 JSON schema.',
          consts: [],
          vars: [],
          funcs: [],
          methods: [],
        },
      ],
      funcs: [{ name: 'OpenGLB', signature: 'func OpenGLB(path string) (*Document, error)', doc: 'OpenGLB reads and decodes a .glb file.' }],
    },
  ],
};

describe('DocsView', () => {
  beforeEach(() => {
    // DocsApp fetches doc.json; return the small index.
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      if (String(input).includes('doc.json')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(DOC_INDEX) } as Response);
      }
      return new Promise<Response>(() => {});
    }) as unknown as typeof fetch;
  });

  it('renders the inline React API reference from the fetched doc.json', async () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(
      screen.getByRole('heading', { level: 2, name: /API documentation/ }),
    ).toBeInTheDocument();

    // DocsApp fetches asynchronously, then renders the package view + symbols.
    expect(await screen.findByRole('heading', { name: /package gltf/ })).toBeInTheDocument();
    expect(container.querySelector('#sym-OpenGLB'), 'func OpenGLB symbol card').not.toBeNull();
    expect(container.querySelector('#sym-Document'), 'type Document symbol card').not.toBeNull();

    // The secondary link to the raw generated static HTML remains.
    expect(screen.getByRole('link', { name: /Open the raw generated HTML/ })).toHaveAttribute('href', './api/');
  });
});
