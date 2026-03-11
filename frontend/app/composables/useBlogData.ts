export interface Comment {
  id: string
  author: {
    name: string
    avatar: string
  }
  content: string
  date: string
}

export interface BlogPost {
  id: string
  title: string
  excerpt: string
  content: string
  author: {
    name: string
    avatar: string
  }
  date: string
  readTime: string
  category: string
  imageUrl: string
  featured?: boolean
  comments: Comment[]
}

export const blogPosts: BlogPost[] = [
  {
    id: '1',
    title: 'The Future of Expressive Design',
    excerpt: 'Exploring how Material You is changing the way we think about personalization and emotion in digital interfaces.',
    content: "Design is no longer just about utility; it's about emotion. Material You represents a paradigm shift in how we approach user interfaces. By extracting colors from a user's wallpaper and applying them across the system, we create a deeply personal experience.\n\nBut Expressive Design goes beyond just color. It's about typography that scales dynamically, shapes that morph based on state, and animations that feel grounded in physics. When a user taps a button, it shouldn't just change color; it should respond with a satisfying ripple, a subtle scale down, and a fluid return to its original state.\n\nIn this article, we'll explore the core principles of Expressive Design and how you can implement them in your next project.",
    author: {
      name: 'Alex Rivera',
      avatar: 'https://picsum.photos/seed/alex/100/100'
    },
    date: 'Oct 12, 2026',
    readTime: '5 min read',
    category: 'Design',
    imageUrl: 'https://picsum.photos/seed/design/800/600',
    featured: true,
    comments: [
      {
        id: 'c1',
        author: { name: 'Jamie Doe', avatar: 'https://picsum.photos/seed/jamie/100/100' },
        content: 'This completely changed my perspective on UI design. The dynamic color extraction is genius.',
        date: 'Oct 13, 2026'
      },
      {
        id: 'c2',
        author: { name: 'Taylor Smith', avatar: 'https://picsum.photos/seed/taylor/100/100' },
        content: "I'm struggling to implement the large border radii without breaking my existing layouts. Any tips?",
        date: 'Oct 14, 2026'
      }
    ]
  },
  {
    id: '2',
    title: 'Building Fluid Animations',
    excerpt: 'A deep dive into creating seamless, physics-based animations that feel natural and responsive.',
    content: "Animations are the soul of an interface. Without them, a digital product feels rigid and lifeless. With them, it feels like a physical object that responds to your touch.\n\nWe'll look at how to use spring physics to create animations that don't just move from A to B, but carry momentum and settle naturally.",
    author: {
      name: 'Sam Taylor',
      avatar: 'https://picsum.photos/seed/sam/100/100'
    },
    date: 'Oct 10, 2026',
    readTime: '8 min read',
    category: 'Development',
    imageUrl: 'https://picsum.photos/seed/animation/800/600',
    comments: []
  },
  {
    id: '3',
    title: 'Color Theory in the Modern Web',
    excerpt: 'How dynamic color algorithms are reshaping brand identities online.',
    content: "Traditionally, a brand had a strict color palette. Today, brands need to be flexible enough to adapt to the user's preferred theme (light/dark) and even their personal color choices.\n\nWe explore the math behind HCT (Hue, Chroma, Tone) and how it ensures accessibility while maintaining vibrant colors.",
    author: {
      name: 'Jordan Lee',
      avatar: 'https://picsum.photos/seed/jordan/100/100'
    },
    date: 'Oct 08, 2026',
    readTime: '6 min read',
    category: 'UX',
    imageUrl: 'https://picsum.photos/seed/color/800/600',
    comments: []
  },
  {
    id: '4',
    title: 'Typography as Interface',
    excerpt: 'When words become the primary interactive elements, how do we ensure usability?',
    content: 'In minimalist design, typography carries the entire weight of the interface. We discuss font pairings, variable fonts, and how to use scale to create clear visual hierarchies without relying on borders or backgrounds.',
    author: {
      name: 'Casey Smith',
      avatar: 'https://picsum.photos/seed/casey/100/100'
    },
    date: 'Oct 05, 2026',
    readTime: '4 min read',
    category: 'Typography',
    imageUrl: 'https://picsum.photos/seed/type/800/600',
    comments: []
  },
  {
    id: '5',
    title: 'The Psychology of Shapes',
    excerpt: 'Why rounded corners and asymmetrical blobs make interfaces feel more approachable.',
    content: 'Humans are biologically wired to perceive sharp edges as a threat and rounded edges as safe. We delve into the psychological reasons behind the shift towards softer, more organic shapes in modern UI design.',
    author: {
      name: 'Alex Rivera',
      avatar: 'https://picsum.photos/seed/alex/100/100'
    },
    date: 'Oct 01, 2026',
    readTime: '7 min read',
    category: 'Psychology',
    imageUrl: 'https://picsum.photos/seed/shapes/800/600',
    comments: []
  }
]

export function useBlogData() {
  const featuredPost = computed(() => blogPosts.find(p => p.featured) ?? blogPosts[0])
  const regularPosts = computed(() => blogPosts.filter(p => !p.featured))

  function getPostById(id: string): BlogPost | undefined {
    return blogPosts.find(p => p.id === id)
  }

  return { blogPosts, featuredPost, regularPosts, getPostById }
}
