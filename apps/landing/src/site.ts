export const site = {
	name: 'Credora',
	tagline: 'Open-source credit decisioning infrastructure.',
	principle: 'Your providers. Your keys. Your decisions.',
	headline: 'Credit decisions, built on your infrastructure.',
	description:
		'Credora is open-source credit decisioning infrastructure for orchestrating providers, policies, scoring, evidence, and decisions.',
	// Replace with the repository URL once it exists. The landing page does not
	// invent a GitHub URL.
	github: '#github',
	docs: '#docs',
};

export type NavLink = { label: string; href: string };

export const navLinks: NavLink[] = [
	{ label: 'Why Credora', href: '#why' },
	{ label: 'How It Works', href: '#how-it-works' },
	{ label: 'Developers', href: '#developers' },
	{ label: 'Architecture', href: '#architecture' },
];