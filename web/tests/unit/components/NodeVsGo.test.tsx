import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { NodeVsGo } from '../../../src/components/NodeVsGo';
import { GLTF } from '../../../src/data';

describe('NodeVsGo', () => {
  it('renders the comparison heading and both JavaScript and Go columns', () => {
    const { container } = render(<NodeVsGo lib={GLTF} />);
    expect(container.querySelector(`#${GLTF.id}-cmp`)).not.toBeNull();
    expect(screen.getByText('JavaScript')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(container.querySelectorAll('.compare .code').length).toBe(2);
  });
});
